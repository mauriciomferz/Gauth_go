import { PoAGraphVisualizer, ProtocolStepVisualizer, generateDemoGraph } from '/static/js/modules/poa-viz.js';

console.log('[PoA] Module script loaded, waiting for DOM...');

let currentVisualizer = null;
let currentMode = 'graph';
window.currentMode = currentMode;

function initVisualizer(mode) {
  console.log('[PoA] initVisualizer called with mode:', mode);
  const vizCanvas = document.getElementById('vizCanvas');
  console.log('[PoA] vizCanvas element:', vizCanvas);
  if (currentVisualizer) {
    console.log('[PoA] Disposing existing visualizer');
    currentVisualizer.dispose();
  }
  currentVisualizer = new PoAGraphVisualizer(vizCanvas);
  console.log('[PoA] Created visualizer:', currentVisualizer);
  console.log('[PoA] currentVisualizer.loadGraph exists?', typeof currentVisualizer.loadGraph);
  window.currentVisualizer = currentVisualizer;
}

function updateStats(stats) {
  document.getElementById('statTotalNodes').textContent = stats.total_nodes || 0;
  document.getElementById('statTotalEdges').textContent = stats.total_edges || 0;
  document.getElementById('statActiveNodes').textContent = stats.active_nodes || 0;
  document.getElementById('statPendingNodes').textContent = stats.pending_nodes || 0;
  document.getElementById('statRevokedNodes').textContent = stats.revoked_nodes || 0;
}

function showInfo(title, description, autoHide = false) {
  document.getElementById('infoTitle').textContent = title;
  document.getElementById('infoDescription').textContent = description;
  const infoPanel = document.getElementById('infoPanel');
  infoPanel.classList.add('visible');
  if (autoHide) {
    setTimeout(() => infoPanel.classList.remove('visible'), 3000);
  }
}

function showLoading(show) {
  const loadingIndicator = document.getElementById('loadingIndicator');
  if (loadingIndicator) loadingIndicator.style.display = show ? 'block' : 'none';
}

function showToast(message, type = 'success') {
  const toast = document.getElementById('toast');
  toast.textContent = message;
  toast.className = `viz-toast ${type}`;
  setTimeout(() => toast.classList.add('show'), 100);
  setTimeout(() => toast.classList.remove('show'), 3000);
}

async function loadGraph() {
  console.log('[PoA] loadGraph called');
  const graphType = document.getElementById('graphType');
  const type = graphType.value;
  let graphData;
  if (type === 'demo') {
    const response = await fetch('/api/v1/visualization/demo/complex-graph', { method: 'POST' });
    const data = await response.json();
    graphData = data.graph;
  } else {
    graphData = generateDemoGraph(type);
  }
  if (!graphData || !graphData.nodes) {
    console.error('[PoA] loadGraph: No graphData returned for type', type);
    showToast('No graph data for ' + type, 'error');
    return;
  }
  await currentVisualizer.loadGraph(graphData);
  updateStats(graphData.stats);
}

async function loadProtocolStep() {
  const protocolStep = document.getElementById('protocolStep');
  const step = protocolStep.value;
  const stepToPattern = {
    'subscription': 'protocol-subscription',
    'matching': 'protocol-matching',
    'request': 'protocol-request',
    'enforcement': 'protocol-enforcement',
    'verification': 'protocol-verification',
    'complete': 'protocol-full'
  };
  const patternName = stepToPattern[step] || 'protocol-subscription';
  const graphData = generateDemoGraph(patternName);
  if (!graphData) throw new Error(`Protocol step '${step}' not found`);
  await currentVisualizer.loadGraph(graphData);
  updateStats(graphData.stats);
}

window.addEventListener('DOMContentLoaded', () => {
  console.log('[PoA] DOM ready, initializing...');
  const vizMode = document.getElementById('vizMode');
  const graphControls = document.getElementById('graphControls');
  const protocolControls = document.getElementById('protocolControls');
  const loadBtn = document.getElementById('loadBtn');
  const clearBtn = document.getElementById('clearBtn');
  const rotateBtn = document.getElementById('rotateBtn');
  const resetBtn = document.getElementById('resetBtn');
  const screenshotBtn = document.getElementById('screenshotBtn');
  const exportBtn = document.getElementById('exportBtn');
  const infoPanel = document.getElementById('infoPanel');
  const closeInfo = document.getElementById('closeInfo');

  console.log('[PoA] DOM Elements Check:', {
    vizCanvas: !!document.getElementById('vizCanvas'),
    vizMode: !!vizMode,
    loadBtn: !!loadBtn,
    clearBtn: !!clearBtn,
    graphType: !!document.getElementById('graphType')
  });

  if (!loadBtn) {
    console.error('[PoA] ERROR: loadBtn is null! Cannot attach event listener.');
    return;
  }

  console.log('[PoA] About to call initVisualizer...');
  initVisualizer('graph');
  console.log('[PoA] After initVisualizer, currentVisualizer is:', currentVisualizer);

  // Mode switcher
  vizMode.addEventListener('change', (e) => {
    currentMode = e.target.value;
    window.currentMode = currentMode;
    if (currentMode === 'graph') {
      graphControls.style.display = 'block';
      protocolControls.style.display = 'none';
    } else {
      graphControls.style.display = 'none';
      protocolControls.style.display = 'block';
    }
    initVisualizer(currentMode);
  });

  // Load button
  console.log('[PoA] Attaching Load button listener to:', loadBtn);
  loadBtn.addEventListener('click', async () => {
    console.log('[PoA] *** Load button clicked - event fired! ***');
    showLoading(true);
    try {
      if (currentMode === 'graph') await loadGraph(); else await loadProtocolStep();
      showInfo('Loaded', 'Visualization loaded successfully', true);
    } catch (error) {
      console.error('Load error:', error);
      showInfo('Error', 'Failed to load visualization: ' + error.message, true);
    } finally { showLoading(false); }
  });

  // Clear button reinforced
  function handleClearClick(e) {
    console.log('[PoA] Clear button handler entered. mode=', currentMode, 'hasVisualizer=', !!currentVisualizer, 'hasClearGraph=', !!currentVisualizer?.clearGraph);
    try {
      if (currentMode === 'graph') {
        if (typeof currentVisualizer?.clearGraph === 'function') {
          console.log('[PoA] Invoking currentVisualizer.clearGraph() from clear button');
          currentVisualizer.clearGraph();
        } else console.warn('[PoA] clearGraph not available on visualizer');
      } else if (currentMode === 'protocol') {
        if (typeof currentVisualizer?.clearVisualization === 'function') {
          console.log('[PoA] Invoking currentVisualizer.clearVisualization() from clear button');
          currentVisualizer.clearVisualization();
        } else console.warn('[PoA] clearVisualization not available for protocol mode');
      }
      updateStats({ total_nodes: 0, total_edges: 0, active_nodes: 0, pending_nodes: 0, revoked_nodes: 0 });
      showInfo('Cleared', 'Visualization cleared', true);
      console.log('[PoA] Clear button handler completed. graphData now =', currentVisualizer?.graphData);
    } catch (err) { console.error('[PoA] Clear button handler error', err); }
  }
  clearBtn.addEventListener('click', handleClearClick);
  clearBtn.onclick = handleClearClick;

  // Rotate button
  let autoRotate = false;
  rotateBtn.addEventListener('click', () => {
    autoRotate = !autoRotate;
    rotateBtn.textContent = autoRotate ? 'Stop Rotate' : 'Rotate';
    if (currentVisualizer.controls) {
      currentVisualizer.controls.autoRotate = autoRotate;
      currentVisualizer.controls.autoRotateSpeed = 2.0;
    }
  });

  // Reset view button
  resetBtn.addEventListener('click', () => {
    if (currentVisualizer.camera) {
      if (currentMode === 'graph') currentVisualizer.camera.position.set(10, 10, 10); else currentVisualizer.camera.position.set(8, 8, 8);
      currentVisualizer.controls.target.set(0, 0, 0);
      currentVisualizer.controls.update();
    }
  });

  // Close info panel
  closeInfo.addEventListener('click', () => infoPanel.classList.remove('visible'));
  setTimeout(() => infoPanel.classList.add('visible'), 500);

  // Fullscreen button
  document.getElementById('fullscreenBtn').addEventListener('click', () => {
    if (!document.fullscreenElement) { document.documentElement.requestFullscreen(); showToast('Entered fullscreen mode'); }
    else { document.exitFullscreen(); showToast('Exited fullscreen mode'); }
  });

  // Screenshot button
  screenshotBtn.addEventListener('click', () => {
    if (currentVisualizer && currentVisualizer.renderer) {
      try {
        const canvas = currentVisualizer.renderer.domElement;
        canvas.toBlob((blob) => {
          const url = URL.createObjectURL(blob);
          const a = document.createElement('a');
          a.href = url; a.download = `poa-visualization-${Date.now()}.png`; a.click();
          URL.revokeObjectURL(url); showToast('Screenshot saved');
        });
      } catch (error) { console.error('Screenshot error:', error); showToast('Failed to capture screenshot', 'error'); }
    }
  });

  // Export button
  exportBtn.addEventListener('click', () => {
    if (currentVisualizer && currentVisualizer.graphData) {
      try {
        const dataStr = JSON.stringify(currentVisualizer.graphData, null, 2);
        const dataBlob = new Blob([dataStr], { type: 'application/json' });
        const url = URL.createObjectURL(dataBlob);
        const a = document.createElement('a');
        a.href = url; a.download = `poa-graph-${Date.now()}.json`; a.click();
        URL.revokeObjectURL(url); showToast('Graph data exported');
      } catch (error) { console.error('Export error:', error); showToast('Failed to export graph data', 'error'); }
    }
  });

  // Track render time
  const originalLoadGraph = currentVisualizer.loadGraph;
  currentVisualizer.loadGraph = async function (...args) {
    const startTime = performance.now();
    const result = await originalLoadGraph.apply(this, args);
    const renderTime = Math.round(performance.now() - startTime);
    document.getElementById('statRenderTime').textContent = `${renderTime}ms`;
    return result;
  };

  // Keyboard shortcuts
  document.addEventListener('keydown', (e) => {
    if (e.key === 'f' && (e.metaKey || e.ctrlKey)) { e.preventDefault(); document.getElementById('fullscreenBtn').click(); }
    else if (e.key === 'r' && (e.metaKey || e.ctrlKey)) { e.preventDefault(); resetBtn.click(); }
    else if (e.key === 'c' && (e.metaKey || e.ctrlKey) && e.shiftKey) { e.preventDefault(); clearBtn.click(); }
  });

  showInfo('Welcome! 🎉 Updated Protocol Flows', '✨ Protocol visualizations now reflect 95% RFC compliance with 30 substeps (was 19). Enhanced: Authorization Chain, Extended Tokens, PIP queries, Commercial Register validation. Use Cmd/Ctrl+F for fullscreen, Cmd/Ctrl+R to reset view.', false);

  console.log('[PoA] Initialization complete!');

  // Auto-load default pattern after initial setup
  setTimeout(async () => {
    try {
      const graphType = document.getElementById('graphType');
      console.log('[PoA] Auto-loading default pattern:', graphType.value);
      await loadGraph();
      showInfo('Default Loaded', 'Initial pattern rendered', true);
    } catch (e) { console.error('[PoA] Auto-load failed', e); showToast('Auto-load failed', 'error'); }
  }, 400);
});
