// toast.js - minimal toast notification queue
let toastContainer;
function ensureContainer(){
  if (!toastContainer){
    toastContainer = document.createElement('div');
    toastContainer.className = 'fixed z-50 top-4 right-4 space-y-2 max-w-sm';
    document.body.appendChild(toastContainer);
  }
}

export function showToast(message, type='info', timeout=4000){
  ensureContainer();
  const el = document.createElement('div');
  const color = type==='error'?'bg-red-600': type==='success'?'bg-green-600': type==='warning'?'bg-yellow-600':'bg-blue-600';
  el.className = `${color} text-white text-sm px-4 py-2 rounded shadow flex items-start gap-2 animate-fade-in`;
  el.innerHTML = `<span class="font-semibold">${type.toUpperCase()}</span><span class="flex-1">${message}</span><button aria-label="Dismiss" class="text-white/70 hover:text-white ml-2">×</button>`;
  const remove=()=>{ el.classList.add('opacity-0','transition-opacity'); setTimeout(()=> el.remove(), 300); };
  el.querySelector('button').addEventListener('click', remove);
  toastContainer.appendChild(el);
  if (timeout>0) setTimeout(remove, timeout);
}

// Basic fade-in keyframes (inject once)
if (!document.getElementById('toast-style')){
  const style=document.createElement('style'); style.id='toast-style';
  style.textContent='@keyframes fadeIn{from{opacity:0;transform:translateY(-4px)}to{opacity:1;transform:translateY(0)}}.animate-fade-in{animation:fadeIn .25s ease-out}';
  document.head.appendChild(style);
}

window.AgentAuth = window.AgentAuth || {}; Object.assign(window.AgentAuth, { showToast });