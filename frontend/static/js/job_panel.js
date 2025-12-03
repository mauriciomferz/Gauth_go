// Extracted from index.html: job panel logic
function fetchJobsAndRender() {
    fetch('/api/v1/beta/examples/run/jobs').catch(()=>fetch('/api/v1/educational/examples/run/jobs'))
        .then(r => r.json())
        .then(data => {
            const jobs = data.jobs || [];
            const tbody = document.getElementById('job-table-body');
            if (!tbody) return;
            if (!jobs.length) {
                tbody.innerHTML = '<tr><td colspan="6" class="text-center text-gray-400 py-4">No jobs found.</td></tr>';
                return;
            }
            tbody.innerHTML = jobs.map(job => {
                const canCancel = job.state === 'running';
                const started = job.started_at ? new Date(job.started_at).toLocaleTimeString() : '-';
                const duration = job.started_at && job.finished_at ?
                    ((new Date(job.finished_at) - new Date(job.started_at))/1000).toFixed(2) + 's'
                    : job.started_at ? ((Date.now() - new Date(job.started_at))/1000).toFixed(2) + 's' : '-';
                return `<tr>
                    <td class="px-3 py-2 font-mono text-xs">${job.id}</td>
                    <td class="px-3 py-2">${job.example_id || '-'}</td>
                    <td class="px-3 py-2">
                        <span class="inline-block px-2 py-1 rounded text-xs font-semibold ${job.state === 'running' ? 'bg-blue-100 text-blue-700' : job.state === 'done' ? 'bg-green-100 text-green-700' : job.state === 'failed' ? 'bg-red-100 text-red-700' : job.state === 'timeout' ? 'bg-yellow-100 text-yellow-700' : 'bg-gray-100 text-gray-700'}">${job.state}</span>
                    </td>
                    <td class="px-3 py-2">${started}</td>
                    <td class="px-3 py-2">${duration}</td>
                    <td class="px-3 py-2">
                        ${canCancel ? `<button class="cancel-job-btn bg-red-500 hover:bg-red-600 text-white px-3 py-1 rounded text-xs" data-job-id="${job.id}"><i class="fas fa-ban"></i> Cancel</button>` : ''}
                    </td>
                </tr>`;
            }).join('');
            // Bind cancel buttons
            tbody.querySelectorAll('.cancel-job-btn').forEach(btn => {
                btn.onclick = function() {
                    const jobId = btn.getAttribute('data-job-id');
                    if (!jobId) return;
                    btn.disabled = true;
                    btn.textContent = 'Cancelling...';
                    fetch(`/api/v1/beta/examples/run/jobs/${jobId}/cancel`, { method: 'POST' }).catch(()=>fetch(`/api/v1/educational/examples/run/jobs/${jobId}/cancel`, { method: 'POST' }))
                        .then(r => r.json())
                        .then(resp => {
                            document.getElementById('job-panel-msg').textContent = resp.message || 'Cancel requested.';
                            setTimeout(() => { document.getElementById('job-panel-msg').textContent = ''; }, 2000);
                            fetchJobsAndRender();
                        })
                        .catch(() => {
                            document.getElementById('job-panel-msg').textContent = 'Cancel failed.';
                            setTimeout(() => { document.getElementById('job-panel-msg').textContent = ''; }, 2000);
                        });
                };
            });
        })
        .catch(() => {
            const tbody = document.getElementById('job-table-body');
            if (tbody) tbody.innerHTML = '<tr><td colspan="6" class="text-center text-red-400 py-4">Failed to load jobs.</td></tr>';
        });
}
// Poll every 2s
setInterval(fetchJobsAndRender, 2000);
document.addEventListener('DOMContentLoaded', fetchJobsAndRender);
