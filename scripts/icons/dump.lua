local exts = {
  "go","ts","tsx","js","jsx","mjs","cjs","json","jsonc","md","markdown","txt","toml","yaml","yml",
  "sql","sh","bash","zsh","fish","lua","vim","py","rb","rs","c","h","cpp","hpp","cc","cs","java",
  "kt","swift","php","ex","exs","erl","hs","ml","zig","odin","nim","dart","scala","clj","pl","r",
  "html","htm","css","scss","sass","less","svelte","vue","astro","graphql","gql","proto","prisma",
  "png","jpg","jpeg","gif","svg","webp","ico","bmp","pdf","mp4","mp3","wav","woff","woff2","ttf","otf",
  "zip","tar","gz","xz","lock","log","env","ini","cfg","conf","diff","patch","csv","tsv","xml","dockerfile",
  "mod","sum","gemfile","makefile","bat","ps1","nix","tf","templ","mdx","edn","gleam","jl","f90","asm","s",
  "wasm","d","v","vala","elm","purs","re","res","cr","sol","pug","ejs","hbs","njk","liquid","twig",
}
local files = {
  "Makefile","Dockerfile","docker-compose.yml","LICENSE","README.md","CHANGELOG.md",".gitignore",
  ".gitattributes",".editorconfig",".env",".env.example","go.mod","go.sum","package.json",
  "package-lock.json","pnpm-lock.yaml","yarn.lock","tsconfig.json","vite.config.ts","Cargo.toml",
  "Cargo.lock","justfile","Justfile",".dockerignore",".prettierrc",".eslintrc",".gitmodules",
  "flake.nix","shell.nix","CMakeLists.txt","build.zig","requirements.txt","pyproject.toml",
}
local dirs = {
  "src","test","tests","docs","doc","lib","bin","build","dist","node_modules","cmd","internal",
  "pkg","api","web","assets","public","config",".github",".git","scripts","migrations","components",
  "hooks","utils","types","styles","images","vendor","examples","tools","server","client","app",
}
local out = { extension = {}, file = {}, directory = {} }
for _, e in ipairs(exts) do
  local g, hl = MiniIcons.get("extension", e)
  out.extension[e] = { glyph = g, hl = hl }
end
for _, f in ipairs(files) do
  local g, hl = MiniIcons.get("file", f)
  out.file[f] = { glyph = g, hl = hl }
end
for _, d in ipairs(dirs) do
  local g, hl = MiniIcons.get("directory", d)
  out.directory[d] = { glyph = g, hl = hl }
end
local dg, dhl = MiniIcons.get("default", "file")
out.default = { glyph = dg, hl = dhl }
local fg, fhl = MiniIcons.get("default", "directory")
out.dirDefault = { glyph = fg, hl = fhl }
io.write(vim.json.encode(out))
