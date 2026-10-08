import type React from 'react';

/**
 * 品牌标识「旋涡 Vortex」v2（docs/brand/ 选定方案，2026-09-10）：
 * 旋涡主标 + 双层对旋虚线环流 + 轨道光点 + 四向漂移风痕 + GOWIND 几何字标签名，
 * 色板同源 docs/design-language.md 主色 hsl(212 100% 45%)。
 * 动效尊重 prefers-reduced-motion；外层不做整体浮动（与慢旋体系不搭），
 * 生命感由光晕呼吸承担。
 * 注：SVG 内嵌 <style> 为文档级作用域，keyframes/类名均带 gw- 前缀防碰撞。
 */
const ANIMATION_CSS = `
.gw-glow{transform-box:view-box;transform-origin:280px 268px;animation:gw-breathe 9s ease-in-out infinite alternate}
.gw-ring1,.gw-ring2,.gw-orbit,.gw-orbit2{transform-box:view-box;transform-origin:280px 268px}
.gw-ring1{animation:gw-spin 48s linear infinite}
.gw-ring2{animation:gw-spin 70s linear infinite reverse}
.gw-orbit{animation:gw-spin 36s linear infinite}
.gw-orbit2{animation:gw-spin 52s linear infinite reverse}
.gw-streak{animation:gw-drift 9s ease-in-out infinite alternate}
.gw-s2{animation-duration:12s;animation-delay:-4s}
.gw-s3{animation-duration:10s;animation-delay:-2s}
.gw-s4{animation-duration:13s;animation-delay:-6s}
@keyframes gw-spin{to{transform:rotate(360deg)}}
@keyframes gw-drift{from{transform:translateX(-16px)}to{transform:translateX(16px)}}
@keyframes gw-breathe{from{transform:scale(1);opacity:1}to{transform:scale(1.05);opacity:0.72}}
@media (prefers-reduced-motion:reduce){
.gw-glow,.gw-ring1,.gw-ring2,.gw-orbit,.gw-orbit2,.gw-streak{animation:none}
}
`;

const SloganIcon: React.FC<React.SVGProps<SVGSVGElement>> = (props) => (
  <svg
    viewBox="0 0 560 560"
    xmlns="http://www.w3.org/2000/svg"
    role="img"
    aria-label="GoWind 风行"
    width="100%"
    height="100%"
    {...props}
  >
    <style>{ANIMATION_CSS}</style>
    <defs>
      <linearGradient
        id="gw-slogan-grad"
        gradientUnits="userSpaceOnUse"
        x1="0"
        y1="38"
        x2="0"
        y2="474"
      >
        <stop offset="0" stopColor="#70B3FF" />
        <stop offset="0.5" stopColor="#0D74F2" />
        <stop offset="1" stopColor="#005AC2" />
      </linearGradient>
      <radialGradient id="gw-slogan-glow" cx="0.5" cy="0.5" r="0.5">
        <stop offset="0" stopColor="#0D74F2" stopOpacity="0.13" />
        <stop offset="1" stopColor="#0D74F2" stopOpacity="0" />
      </radialGradient>
    </defs>
    <circle className="gw-glow" cx="280" cy="268" r="252" fill="url(#gw-slogan-glow)" />
    <circle
      className="gw-ring2"
      cx="280"
      cy="268"
      r="246"
      fill="none"
      stroke="#0D74F2"
      strokeOpacity="0.07"
      strokeWidth="1.5"
      strokeDasharray="2 16"
      strokeLinecap="round"
    />
    <circle
      className="gw-ring1"
      cx="280"
      cy="268"
      r="212"
      fill="none"
      stroke="#0D74F2"
      strokeOpacity="0.14"
      strokeWidth="1.5"
      strokeDasharray="3 13"
      strokeLinecap="round"
    />
    <g className="gw-orbit">
      <circle cx="280" cy="113" r="4.5" fill="#3B82F6" opacity="0.55" />
      <circle cx="280" cy="423" r="3" fill="#70B3FF" opacity="0.45" />
    </g>
    <g className="gw-orbit2">
      <circle cx="435" cy="268" r="3.5" fill="#0D74F2" opacity="0.4" />
    </g>
    <g
      className="gw-streak"
      fill="none"
      stroke="#0D74F2"
      strokeOpacity="0.12"
      strokeWidth="10"
      strokeLinecap="round"
    >
      <path d="M52 190 H140 A20 20 0 0 0 120 156" />
    </g>
    <g
      className="gw-streak gw-s2"
      fill="none"
      stroke="#0D74F2"
      strokeOpacity="0.1"
      strokeWidth="10"
      strokeLinecap="round"
    >
      <path d="M400 126 H494 A22 22 0 0 0 474 88" />
    </g>
    <g
      className="gw-streak gw-s3"
      fill="none"
      stroke="#0D74F2"
      strokeOpacity="0.12"
      strokeWidth="10"
      strokeLinecap="round"
    >
      <path d="M368 440 H480 A24 24 0 0 0 460 402" />
    </g>
    <g
      className="gw-streak gw-s4"
      fill="none"
      stroke="#0D74F2"
      strokeOpacity="0.09"
      strokeWidth="10"
      strokeLinecap="round"
    >
      <path d="M84 465 H161 A18 18 0 0 0 143 435" />
    </g>
    <g
      transform="translate(172,160) scale(0.42)"
      fill="none"
      stroke="url(#gw-slogan-grad)"
      strokeLinecap="round"
    >
      {/* 旋涡本体缓旋：24s/圈、与环流同向；SMIL 以字形局部坐标 (256,256) 为轴，避免嵌套变换下的 origin 歧义 */}
      <g>
        <animateTransform
          attributeName="transform"
          type="rotate"
          from="0 256 256"
          to="360 256 256"
          dur="24s"
          repeatCount="indefinite"
        />
        <path d="M241 431 A176 176 0 1 1 408 344" strokeWidth="84" />
        <path d="M264 199 A58 58 0 1 1 204 282" strokeWidth="54" />
      </g>
    </g>
    <g
      transform="translate(185,464) scale(0.3)"
      fill="none"
      stroke="#0D74F2"
      strokeOpacity="0.5"
      strokeWidth="20"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <path d="M70 22 A40 40 0 1 0 82 50 L56 50" />
      <g transform="translate(126,0)">
        <circle cx="42" cy="50" r="40" />
      </g>
      <path transform="translate(254,0)" d="M10 0 L30 100 L48 30 L66 100 L86 0" />
      <path transform="translate(384,0)" d="M10 0 L10 100" />
      <path transform="translate(438,0)" d="M10 100 L10 0 L74 100 L74 0" />
      <path
        transform="translate(546,0)"
        d="M10 0 L10 100 M10 0 L24 0 A50 50 0 0 1 24 100 L10 100"
      />
    </g>
  </svg>
);

export default SloganIcon;
