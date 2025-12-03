// refresh.js - shared exponential backoff JSON fetch helper
export async function backoffFetchJSON(url, attempt=1){
  const maxDelay = 4000;
  try {
    const res = await fetch(url, {headers:{'Accept':'application/json'}});
    // Silently fail on 404 - API endpoint doesn't exist
    if (res.status === 404) {
      return null;
    }
    if (!res.ok) {
      throw new Error(`HTTP ${res.status}`);
    }
    return await res.json();
  } catch(err){
    const delay = Math.min(maxDelay, Math.pow(2, attempt) * 125);
    await new Promise(r=>setTimeout(r, delay));
    if(attempt < 5) return backoffFetchJSON(url, attempt+1); else throw err;
  }
}
