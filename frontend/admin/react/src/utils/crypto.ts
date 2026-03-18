import CryptoJS from 'crypto-js';

/**
 * 使用 AES 加密
 * @param data 要加密的数据
 * @param key 密钥
 * @returns 加密后的字符串
 */
export function encryptByAES(data: string, key: string): string {
    try {
        const keyHex = CryptoJS.enc.Utf8.parse(key);
        const ivHex = CryptoJS.enc.Utf8.parse(key);
        const encrypted = CryptoJS.AES.encrypt(data, keyHex, {
            iv: ivHex,
            mode: CryptoJS.mode.CBC,
            padding: CryptoJS.pad.Pkcs7,
        });
        return encrypted.toString();
    } catch (error) {
        console.error('AES encryption failed:', error);
        throw error;
    }
}

/**
 * 使用 AES 解密
 * @param encryptedData 加密的字符串
 * @param key 密钥
 * @returns 解密后的原始数据
 */
export function decryptByAES(encryptedData: string, key: string): string {
    try {
        const decrypted = CryptoJS.AES.decrypt(encryptedData, CryptoJS.enc.Utf8.parse(key), {
            mode: CryptoJS.mode.ECB,
            padding: CryptoJS.pad.Pkcs7,
        });
        return decrypted.toString(CryptoJS.enc.Utf8);
    } catch (error) {
        console.error('AES decryption failed:', error);
        throw error;
    }
}

/**
 * 使用 MD5 加密
 * @param data 要加密的数据
 * @returns MD5 哈希值（32 位小写）
 */
export function encryptByMD5(data: string): string {
    try {
        return CryptoJS.MD5(data).toString();
    } catch (error) {
        console.error('MD5 encryption failed:', error);
        throw error;
    }
}

/**
 * 使用 SHA256 加密
 * @param data 要加密的数据
 * @returns SHA256 哈希值
 */
export function encryptBySHA256(data: string): string {
    try {
        return CryptoJS.SHA256(data).toString();
    } catch (error) {
        console.error('SHA256 encryption failed:', error);
        throw error;
    }
}

/**
 * 使用 Base64 编码
 * @param data 要编码的数据
 * @returns Base64 编码后的字符串
 */
export function encodeBase64(data: string): string {
    try {
        return CryptoJS.enc.Base64.stringify(CryptoJS.enc.Utf8.parse(data));
    } catch (error) {
        console.error('Base64 encoding failed:', error);
        throw error;
    }
}

/**
 * 使用 Base64 解码
 * @param base64Data Base64 编码的字符串
 * @returns 解码后的原始数据
 */
export function decodeBase64(base64Data: string): string {
    try {
        return CryptoJS.enc.Base64.parse(base64Data).toString(CryptoJS.enc.Utf8);
    } catch (error) {
        console.error('Base64 decoding failed:', error);
        throw error;
    }
}

/**
 * 生成随机密钥
 * @param length 密钥长度（字节）
 * @returns 随机密钥字符串（Hex 格式）
 */
export function generateSecretKey(length: number = 32): string {
    try {
        const randomArray = new Uint8Array(length);
        crypto.getRandomValues(randomArray);
        return Array.from(randomArray)
            .map(b => b.toString(16).padStart(2, '0'))
            .join('');
    } catch (error) {
        console.error('Secret key generation failed:', error);
        throw error;
    }
}

/**
 * 生成 HMAC-SHA256 签名
 * @param data 要签名的数据
 * @param secret 密钥
 * @returns HMAC-SHA256 签名
 */
export function generateHMAC(data: string, secret: string): string {
    try {
        const hmac = CryptoJS.HmacSHA256(data, secret);
        return hmac.toString();
    } catch (error) {
        console.error('HMAC generation failed:', error);
        throw error;
    }
}

/**
 * 验证 HMAC 签名
 * @param data 原始数据
 * @param signature 签名
 * @param secret 密钥
 * @returns 签名是否有效
 */
export function verifyHMAC(data: string, signature: string, secret: string): boolean {
    try {
        const expectedSignature = generateHMAC(data, secret);
        return expectedSignature === signature;
    } catch (error) {
        console.error('HMAC verification failed:', error);
        return false;
    }
}

export default {
    encryptByAES,
    decryptByAES,
    encryptByMD5,
    encryptBySHA256,
    encodeBase64,
    decodeBase64,
    generateSecretKey,
    generateHMAC,
    verifyHMAC,
};
