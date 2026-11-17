"use strict";

const TEXT = {
  en: ["Reasonix Studio is starting", "Waiting for the local service to be ready. The workspace opens here as soon as it is."],
  zh: ["Reasonix Studio 正在启动", "正在等待本地服务就绪，就绪后会在此窗口打开工作区。"],
};

function startingPage(locale) {
  const [title, note] = String(locale).toLowerCase().startsWith("zh") ? TEXT.zh : TEXT.en;
  return `<!doctype html><meta charset="utf-8"><title>${title}</title><style>
:root{color-scheme:light dark;--bg:#fafafa;--fg:#1c1c1e;--mute:#6b6b70}
@media(prefers-color-scheme:dark){:root{--bg:#1b1b1d;--fg:#ececee;--mute:#9a9aa0}}
html,body{height:100%}
body{margin:0;display:grid;place-items:center;background:var(--bg);color:var(--fg);font:14px system-ui,sans-serif}
.launch{width:min(420px,calc(100vw - 56px))}
h1{margin:0 0 8px;font-size:17px;font-weight:600}
p{margin:0;color:var(--mute);line-height:1.5}
.bar{height:3px;margin-top:18px;border-radius:2px;background:var(--mute);opacity:.35;overflow:hidden;position:relative}
.bar::after{content:"";position:absolute;inset:0 70% 0 0;background:var(--fg);animation:run 1.4s ease-in-out infinite}
@keyframes run{from{transform:translateX(-100%)}to{transform:translateX(340%)}}
</style><div class="launch"><h1>${title}</h1><p>${note}</p><div class="bar"></div></div>`;
}

module.exports = { startingPage };
