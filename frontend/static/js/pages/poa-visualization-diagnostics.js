// Diagnostics overlay: logs button clicks & current visualizer state
(function(){
  const ids=['loadBtn','clearBtn','rotateBtn','resetBtn','screenshotBtn','exportBtn'];
  function logState(label){
    const viz=window.currentVisualizer;
    console.log('[PoA][Diag]', label, {
      hasViz: !!viz,
      hasControls: !!viz?.controls,
      autoRotate: !!viz?.controls?.autoRotate,
      graphNodes: viz?.nodes?.size || 0,
      graphEdges: viz?.edges?.length || 0,
      hasGraphData: !!viz?.graphData,
      mode: window?.currentMode || 'unknown'
    });
  }
  ids.forEach(id=>{ const el=document.getElementById(id); if(el){ el.addEventListener('click',()=>logState('CLICK '+id), {capture:true}); } });
  let beats=0; const hb=setInterval(()=>{ if(beats++>20){ clearInterval(hb); return; } logState('HEARTBEAT'); },1500);
  console.log('[PoA][Diag] diagnostics overlay initialized');
})();
