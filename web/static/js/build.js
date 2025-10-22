// Bundles all JS modules into bundle.js for production use (ESM)
import * as esbuild from "esbuild";

esbuild.build({
  entryPoints: [
    "web/static/js/modules/main.js"
  ],
  bundle: true,
  minify: true,
  sourcemap: true,
  outfile: "web/static/js/bundle.js",
  format: "iife",
  target: ["es2021"],
  platform: "browser"
}).catch(() => process.exit(1));
