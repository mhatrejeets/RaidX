const fs = require("fs");
const path = require("path");

const root = path.resolve(__dirname, "..");
const mainPath = path.resolve(root, "..", "main.go");
const targets = [
  path.resolve(root, "apps", "web", "src", "App.jsx"),
  path.resolve(root, "apps", "mobile", "src", "App.js"),
  path.resolve(root, "apps", "web", "src", "authClient.js"),
  path.resolve(root, "apps", "mobile", "src", "authClient.js"),
  path.resolve(root, "packages", "shared", "src", "authClient.js"),
  path.resolve(root, "packages", "shared", "src", "routes.js")
];

const main = fs.readFileSync(mainPath, "utf8");
const routeRe = /app\.(Get|Post|Put|Delete)\("([^"]+)"/g;
const routes = [...new Set([...main.matchAll(routeRe)].map((m) => m[2]))].sort();

let text = "";
for (const target of targets) {
  if (fs.existsSync(target)) {
    text += "\n" + fs.readFileSync(target, "utf8");
  }
}

const strRe = /(["'`])([^"'`]*\/[^"]*?)\1/g;
const strings = [...new Set([...text.matchAll(strRe)].map((m) => m[2]))];

function escapeRegex(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

function covers(route) {
  const pattern = "^" + escapeRegex(route).replace(/:[^/]+/g, "[^/]+") + "$";
  const regex = new RegExp(pattern);
  return strings.some((value) => regex.test(value));
}

const remaining = routes.filter((route) => !covers(route));

console.log(`TOTAL ${routes.length}`);
console.log(`COVERED ${routes.length - remaining.length}`);
console.log(`REMAINING ${remaining.length}`);

if (remaining.length) {
  for (const route of remaining) {
    console.log(route);
  }
  process.exitCode = 1;
}
