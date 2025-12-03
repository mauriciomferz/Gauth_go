// Extracted from index.html: log streaming panel logic
(function(){
    const jobIdInput = document.getElementById('logJobId');
    const startBtn = document.getElementById('startLogStream');
    const stopBtn = document.getElementById('stopLogStream');
    const outEl = document.getElementById('logOutput');
    const statusEl = document.getElementById('logStatus');
    let es = null;

    function append(line){
        if(!line) return;
        if(outEl.firstChild && outEl.firstChild.nodeType === Node.TEXT_NODE){
            outEl.textContent += "\n" + line;
        } else {
            outEl.innerText += (outEl.innerText.trim()==='// Logs will appear here...' ? '' : '\n') + line;
        }
        outEl.scrollTop = outEl.scrollHeight;
    }

    startBtn.addEventListener('click', () => {
        const id = jobIdInput.value.trim();
        if(!id){
            statusEl.textContent = 'Please provide a job ID first.';
            return;
        }
        if(es){ es.close(); }
        statusEl.textContent = 'Connecting...';
        let primary = `/api/v1/beta/examples/run/${encodeURIComponent(id)}/logs`;
        let fallback = `/api/v1/educational/examples/run/${encodeURIComponent(id)}/logs`;
        function openStream(url, triedFallback){
            es = new EventSource(url);
            es.onerror = () => {
                if(!triedFallback){
                    es.close();
                    statusEl.textContent = 'Primary beta path failed, trying deprecated path...';
                    openStream(fallback, true);
                }
            };
        }
        openStream(primary, false);
        startBtn.disabled = true; stopBtn.disabled = false;
        outEl.innerText = '';
        es.addEventListener('open', () => {
            statusEl.textContent = 'Connected – awaiting first events';
        });
        es.addEventListener('status', ev => {
            statusEl.textContent = 'Status: ' + ev.data;
            try { const obj = JSON.parse(ev.data); if (obj.state) append('[status] '+obj.state); } catch {}
        });
        es.addEventListener('log', ev => {
            append(ev.data);
            statusEl.textContent = 'Receiving log output...';
        });
        es.addEventListener('done', ev => {
            append('[END OF LOG STREAM]');
            statusEl.textContent = 'Stream complete';
            es.close(); es = null; startBtn.disabled = false; stopBtn.disabled = true;
        });
        es.onerror = (e) => {
            statusEl.textContent = 'Stream error or closed.';
            if(es){ es.close(); es = null; }
            startBtn.disabled = false; stopBtn.disabled = true;
        };
    });

    stopBtn.addEventListener('click', () => {
        if(es){ es.close(); es = null; }
        statusEl.textContent = 'Stream stopped by user';
        startBtn.disabled = false; stopBtn.disabled = true;
    });
})();
