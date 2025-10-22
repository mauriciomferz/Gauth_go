// refresh.js - shared exponential backoff JSON fetch helper
export async function backoffFetchJSON(url, attempt=1){
  const maxDelay = 4000;
  try {
    const res = await fetch(url, {headers:{'Accept':'application/json'}});
    return await res.json();
  } catch(err){
    const delay = Math.min(maxDelay, Math.pow(2, attempt) * 125);
    await new Promise(r=>setTimeout(r, delay));
    if(attempt < 5) return backoffFetchJSON(url, attempt+1); else throw err;
  }
}
