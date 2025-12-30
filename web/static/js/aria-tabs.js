// Accessible tab behavior for Interactive Demo (minimal; replace with richer implementation later)
(function(){
  function initTabs(){
    const buttons = document.querySelectorAll('button[data-tab]');
    if(!buttons.length) return;
    buttons.forEach(btn => {
      btn.setAttribute('role','tab');
      btn.setAttribute('tabindex', btn.classList.contains('active') ? '0' : '-1');
      btn.addEventListener('click', () => activate(btn.getAttribute('data-tab')));
      btn.addEventListener('keydown', e => {
        if(['ArrowRight','ArrowLeft','Home','End'].includes(e.key)){
          e.preventDefault();
          const list = Array.from(buttons);
            let idx = list.indexOf(btn);
            if(e.key==='ArrowRight') idx = (idx+1)%list.length; else if(e.key==='ArrowLeft') idx = (idx-1+list.length)%list.length; else if(e.key==='Home') idx = 0; else if(e.key==='End') idx = list.length-1;
            list[idx].focus();
        }
      });
    });
    function activate(id){
      buttons.forEach(b=>{b.classList.toggle('active', b.getAttribute('data-tab')===id); b.setAttribute('tabindex', b.classList.contains('active')?'0':'-1');});
      document.querySelectorAll('.tab-content').forEach(p=>{p.classList.toggle('active', p.id===id); p.classList.toggle('hidden', p.id!==id);});
    }
  }
  if(document.readyState==='loading') document.addEventListener('DOMContentLoaded', initTabs); else initTabs();
})();// Lightweight ARIA tabs enhancement (kept separate to avoid interfering with legacy/app.js duplication)
(function(){
  document.addEventListener('DOMContentLoaded', function(){
    const tabNav = document.querySelector('nav[aria-label="Interactive demo tabs"]');
    if(!tabNav) return;
    const tabs = Array.from(tabNav.querySelectorAll('.tab-button[data-tab]'));
    if(!tabs.length) return;
    tabNav.setAttribute('role','tablist');
    const panels = tabs.map(t => document.getElementById(t.getAttribute('data-tab'))).filter(Boolean);
    tabs.forEach((t,i)=>{
      if(!t.id) t.id = 'demo-tab-'+i;
      t.setAttribute('role','tab');
      t.setAttribute('tabindex','-1');
      t.setAttribute('aria-selected','false');
    });
    panels.forEach(p=>{
      p.setAttribute('role','tabpanel');
      p.setAttribute('tabindex','0');
      const owner = tabs.find(tb => tb.getAttribute('data-tab') === p.id);
      if(owner) p.setAttribute('aria-labelledby', owner.id);
      if(!p.classList.contains('active') p.setAttribute('hidden','hidden');
    });
    function activate(tab){
      const target = tab.getAttribute('data-tab');
      tabs.forEach(tb => {
        const sel = tb === tab;
        tb.classList.toggle('active', sel);
        tb.setAttribute('aria-selected', sel? 'true':'false');
        tb.setAttribute('tabindex', sel? '0':'-1');
      });
      panels.forEach(p => {
        const show = p.id === target;
        if(show){
          p.classList.add('active');
          p.style.display='block';
          p.removeAttribute('hidden');
        } else {
          p.classList.remove('active');
            p.style.display='none';
            p.setAttribute('hidden','hidden');
        }
      });
    }
    const initial = tabs.find(t=>t.classList.contains('active') || tabs[0];
    if(initial) activate(initial);
    tabs.forEach(tab => {
      tab.addEventListener('click', e => { e.preventDefault(); activate(tab); tab.focus(); });
      tab.addEventListener('keydown', e => {
        const i = tabs.indexOf(tab); let ni = -1;
        switch(e.key){
          case 'ArrowRight': ni=(i+1)%tabs.length; break;
          case 'ArrowLeft': ni=(i-1+tabs.length)%tabs.length; break;
          case 'Home': ni=0; break;
          case 'End': ni=tabs.length-1; break;
          default: return;
        }
        e.preventDefault();
        const nt = tabs[ni]; activate(nt); nt.focus();
      });
    });
    console.log('[ARIA] Tabs initialized');
  });
})();
