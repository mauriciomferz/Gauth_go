// Console output utilities (initial modularization)
export function addConsoleOutput(containerId, message, type = "info") {
  const container = document.getElementById(containerId);
  if (!container) return;
  const timestamp = new Date().toISOString().split("T")[1].split(".")[0];
  const typeClass = type === "success" ? "console-success"
    : type === "error" ? "console-error"
    : type === "warning" ? "console-warning"
    : "console-info";
  const line = document.createElement("div");
  line.className = `console-line ${typeClass}`;
  line.innerHTML = `<span class="text-gray-400">[${timestamp}]</span> ${message}`;
  container.appendChild(line);
  container.scrollTop = container.scrollHeight;
}

export function clearConsoleOutput(containerId) {
  const container = document.getElementById(containerId);
  if (!container) return;
  container.innerHTML = `
    <span class="text-gray-500"># ${containerId.replace("-output", "").toUpperCase()} Console</span><br>
    <span class="text-blue-400">gauth-${containerId.replace("-output", "")}></span> <span class="blinking-cursor">_</span>
  `;
}

export function escapeHtml(str) {
  if (typeof str !== "string") return "";
  return str.replace(/[&<>"']/g, c => ({"&":"&amp;","<":"&lt;",">":"&gt;","\"":"&quot;","'":"&#39;"}[c]||c));
}
