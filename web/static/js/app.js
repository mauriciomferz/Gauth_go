// app.js legacy shim (legacy monolith removed)
// All real functionality lives in ES modules under /static/js/modules/main.js
// This file only forwards old global names; delete once templates stop using them.
(function(){
const LEGACY = [
"createToken","validateToken","revokeToken",
"checkAuthorization","publishEvent","subscribeEvents",
"viewAuditLog","generateReport",
"runAdvancedSuite","runAllBasics","runSample"
];
function wire(){
if(!window.GAuth) return false;
LEGACY.forEach(n=>{
window[n] = function(){
const fn = window.GAuth && window.GAuth[n];
if (typeof fn === "function") return fn.apply(window.GAuth, arguments);
console.warn("[app.js shim] missing", n);
};
});
window.scrollToDemo = function(){
document.getElementById("demo")?.scrollIntoView({behavior:"smooth"});
};
console.info("[app.js shim] legacy forwards attached");
return true;
}
if(!wire()){
const h = setInterval(()=>{ if(wire()) clearInterval(h); }, 50);
setTimeout(()=>clearInterval(h), 5000);
}
})();