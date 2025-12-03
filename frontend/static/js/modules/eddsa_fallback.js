// eddsa_fallback.js - Ed25519 verification using tweetnacl (public domain). Provides constant-time-ish detached signature verification.
// If WebCrypto Ed25519 is unavailable, this module offers a reliable fallback.
// API: verifyEd25519(publicKeyUint8, messageUint8, signatureUint8) => boolean

let naclLib = null;
function ensureLib(){
  if(naclLib) return naclLib;
  // Try CommonJS require first (Jest / Node)
  try { // eslint-disable-next-line global-require, import/no-extraneous-dependencies
    naclLib = require('tweetnacl');
    if(naclLib && naclLib.default && !naclLib.sign){ naclLib = naclLib.default; }
    return naclLib;
  } catch(e) {}
  // Try global (Playwright/browser preload)
  if(typeof window !== 'undefined' && window.nacl){ naclLib = window.nacl; return naclLib; }
  return naclLib; // may be null
}

export function setEd25519Impl(lib){
  if(lib && lib.sign && lib.sign.detached && lib.sign.detached.verify){
    naclLib = lib;
  }
}

export function verifyEd25519(pub, msg, sig){
  try {
    if(!pub || !sig || !msg) return false;
    if(pub.length !== 32 || sig.length !== 64) return false;
    const lib = ensureLib();
    if(lib && lib.sign && lib.sign.detached && lib.sign.detached.verify){
      return lib.sign.detached.verify(msg, sig, pub);
    }
    return false;
  } catch(err){ return false; }
}
