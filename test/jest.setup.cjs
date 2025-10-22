// Polyfills for Node test environment
const { TextEncoder, TextDecoder } = require('util');
if(!global.TextEncoder) global.TextEncoder = TextEncoder;
if(!global.TextDecoder) global.TextDecoder = TextDecoder;
// Minimal TransformStream polyfill (NOT streaming correct semantics, only structural placeholder)
if(!global.TransformStream){
  class TransformStreamPolyfill {
    constructor(){
      const { PassThrough } = require('stream');
      this.readable = new PassThrough();
      this.writable = this.readable;
    }
  }
  global.TransformStream = TransformStreamPolyfill;
}
