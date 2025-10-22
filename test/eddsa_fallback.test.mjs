import { expect, test } from '@jest/globals';
import { verifyEd25519, setEd25519Impl } from '../web/static/js/modules/eddsa_fallback.js';
import nacl from 'tweetnacl';

function u8(arr){ return new Uint8Array(arr); }

test('verifyEd25519 returns false on invalid lengths', () => {
  expect(verifyEd25519(u8([1,2,3]), u8([4]), u8([5]))).toBe(false);
});

test('verifyEd25519 verifies a valid signature', () => {
  const keyPair = nacl.sign.keyPair();
  const msg = new Uint8Array([116,101,115,116,45,109,101,115,115,97,103,101]); // 'test-message'
  const sig = nacl.sign.detached(msg, keyPair.secretKey);
  expect(nacl.sign.detached.verify(msg, sig, keyPair.publicKey)).toBe(true);
  setEd25519Impl(nacl); // inject reliable implementation
  expect(verifyEd25519(keyPair.publicKey, msg, sig)).toBe(true);
});

test('verifyEd25519 fails for modified signature', () => {
  const keyPair = nacl.sign.keyPair();
  const msg = new Uint8Array([97,110,111,116,104,101,114,45,109,101,115,115,97,103,101]); // 'another-message'
  const sig = nacl.sign.detached(msg, keyPair.secretKey);
  sig[0] ^= 0xff; // flip a bit
  expect(verifyEd25519(keyPair.publicKey, msg, sig)).toBe(false);
});
