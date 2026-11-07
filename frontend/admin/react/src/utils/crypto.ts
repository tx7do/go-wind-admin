import CryptoJS from 'crypto-js';

/**
 * 传输加密工具：与登录同规的 AES-128-CBC（key = iv = VITE_AES_KEY，PKCS7，base64 输出）。
 * 后端对登录/注册/改密/重置密码均以 NeedDecrypt 解密，前端必须密文传输。
 */

function encryptData(data: string, key: string, iv: string): string {
  const keyHex = CryptoJS.enc.Utf8.parse(key);
  const ivHex = CryptoJS.enc.Utf8.parse(iv);
  const encrypted = CryptoJS.AES.encrypt(data, keyHex, {
    iv: ivHex,
    mode: CryptoJS.mode.CBC,
    padding: CryptoJS.pad.Pkcs7,
  });
  return encrypted.toString();
}

export function encryptPassword(password: string): string {
  const key = import.meta.env.VITE_AES_KEY;
  if (!key) {
    throw new Error('VITE_AES_KEY is not set in environment');
  }
  return encryptData(password, key, key);
}

/**
 * P1-12 B 路线：口令传输混合信封（WebCrypto，与后端 pkg/passwordcrypto 同格式）。
 *
 * 格式 "rsa-v1:<b64(wrappedKey)>:<b64(nonce)>:<b64(cipher)>":
 * 随机 AES-256-GCM 会话密钥加密口令，RSA-2048 OAEP(SHA-256) 包裹会话密钥。
 * 公钥静态内置（公开值不降低安全性，无分发/替换通道）；私钥仅服务端持有。
 * 服务端兼容旧 base64+AES 形态（rsa-v1 前缀优先识别），过渡期两格式并存。
 */
const PASSWORD_RSA_PUB_B64 = import.meta.env.VITE_PASSWORD_RSA_PUB ?? '';

function base64ToBytes(b64: string): Uint8Array {
  const bin = atob(b64);
  const bytes = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
  return bytes;
}

function bytesToBase64(bytes: Uint8Array): string {
  let bin = '';
  for (let i = 0; i < bytes.length; i++) bin += String.fromCharCode(bytes[i]);
  return btoa(bin);
}

/** 解析 SPKI DER：容忍完整 PEM 头尾与换行空白，统一为裸 base64 再解码。 */
function parseSpkiDer(b64: string): ArrayBuffer {
  const compact = b64
    .replace(/-----BEGIN PUBLIC KEY-----/g, '')
    .replace(/-----END PUBLIC KEY-----/g, '')
    .replace(/\s+/g, '');
  return base64ToBytes(compact).slice().buffer as ArrayBuffer;
}

/** 信封加密。公钥未配置时抛错（构建期必须注入 VITE_PASSWORD_RSA_PUB）。 */
export async function encryptPasswordEnvelope(password: string): Promise<string> {
  if (!PASSWORD_RSA_PUB_B64) {
    throw new Error('VITE_PASSWORD_RSA_PUB is not set in environment');
  }
  const der = parseSpkiDer(PASSWORD_RSA_PUB_B64);
  // RSA 公钥只能以 'spki' 格式导入：'raw' 仅为对称密钥格式，
  // 传入 RSA-OAEP 会抛 Unsupported import key format for algorithm。
  const key = await crypto.subtle.importKey(
    'spki', der,
    { name: 'RSA-OAEP', hash: 'SHA-256' },
    false, ['encrypt'],
  );

  // 会话密钥必须声明为可导出（extractable=true）：随后要 exportKey('raw')
  // 取出原始字节交 RSA 包裹；false 会导致 exportKey 抛 key is not extractable。
  const aesKey = crypto.subtle.generateKey({ name: 'AES-GCM', length: 256 }, true, ['encrypt']);
  const keyObj = await aesKey;
  const nonce = crypto.getRandomValues(new Uint8Array(12));
  const cipher = await crypto.subtle.encrypt({ name: 'AES-GCM', iv: nonce }, keyObj, new TextEncoder().encode(password));
  const rawAesKey = await crypto.subtle.exportKey('raw', keyObj);
  const wrapped = await crypto.subtle.encrypt({ name: 'RSA-OAEP' }, key, rawAesKey);

  return (
    'rsa-v1:' + bytesToBase64(new Uint8Array(wrapped)) + ':' +
    bytesToBase64(nonce) + ':' +
    bytesToBase64(new Uint8Array(cipher))
  );
}
