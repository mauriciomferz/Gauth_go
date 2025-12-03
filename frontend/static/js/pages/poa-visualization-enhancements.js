// Enhancement layer: guards & user feedback for premature actions
(function(){
  function toast(msg, type){
    const t=document.getElementById('toast');
    if(!t) return;
    t.textContent=msg; t.className='viz-toast '+(type||'success');
    setTimeout(()=>t.classList.add('show'),50);
    setTimeout(()=>t.classList.remove('show'),2500);
  }
  function attach(){
    const rotateBtn=document.getElementById('rotateBtn');
    const exportBtn=document.getElementById('exportBtn');
    const screenshotBtn=document.getElementById('screenshotBtn');
    const clearBtn=document.getElementById('clearBtn');
    const resetBtn=document.getElementById('resetBtn');
    const viz=window.currentVisualizer;
    if(!viz){ return false; }
    if(exportBtn && !exportBtn._enhanced){
      exportBtn.addEventListener('click', function(e){
        if(!viz.graphData){ e.stopImmediatePropagation(); toast('Load a graph first','error'); }
      }, true);
      exportBtn._enhanced=true;
    }
    if(screenshotBtn && !screenshotBtn._enhanced){
      screenshotBtn.addEventListener('click', function(e){
        if(!viz.renderer){ e.stopImmediatePropagation(); toast('Renderer not ready','error'); return; }
        setTimeout(()=>{
          if(!e.defaultPrevented){
            try {
              const canvas=viz.renderer.domElement;
              if(!canvas) throw new Error('Canvas missing');
              const url=canvas.toDataURL('image/png');
              if(url && url.length>50){ const a=document.createElement('a'); a.href=url; a.download='poa-visualization-fallback.png'; a.click(); toast('Screenshot saved'); }
            } catch(err){ console.warn('[PoA] Fallback screenshot error', err); }
          }
        },10);
      }, false);
      screenshotBtn._enhanced=true;
    }
    if(rotateBtn && !rotateBtn._enhanced){
      rotateBtn.addEventListener('click', ()=>{ if(!viz.controls){ toast('Controls not ready','error'); } });
      rotateBtn._enhanced=true;
    }
    if(resetBtn && !resetBtn._enhanced){
      resetBtn.addEventListener('click', ()=>{ if(!viz.camera){ toast('Camera not ready','error'); } });
      resetBtn._enhanced=true;
    }
    if(clearBtn && !clearBtn._enhanced){
      clearBtn.addEventListener('click', ()=>{ if(!viz){ toast('Visualizer not ready','error'); } });
      clearBtn._enhanced=true;
    }
    return true;
  }
  let attempts=0; const iv=setInterval(()=>{ if(attach()||attempts++>50){ clearInterval(iv); } },200);
})();
