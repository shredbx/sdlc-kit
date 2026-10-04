#!/usr/bin/env bash
set -euo pipefail

SERVICE_PATH="consumers/clients/$INPUT_SERVICE_CODENAME/$INPUT_SERVICE_REPO_NAME/projects/$INPUT_SERVICE_PROJECT_NAME/$INPUT_SERVICE_SERVICE_NAME"

# Create service folder
mkdir -p "$SERVICE_PATH"
mkdir -p "$SERVICE_PATH/src"
mkdir -p "$SERVICE_PATH/src/components"
mkdir -p "$SERVICE_PATH/src/lib"
mkdir -p "$SERVICE_PATH/public"
mkdir -p "$SERVICE_PATH/nginx"

# Create service record
SERVICE_RECORD_DIR="processos-workspace/records/clients/service"
mkdir -p "$SERVICE_RECORD_DIR"
SERVICE_RECORD_FILE="$SERVICE_RECORD_DIR/$INPUT_SERVICE_CODENAME-$INPUT_SERVICE_REPO_NAME-$INPUT_SERVICE_PROJECT_NAME-$INPUT_SERVICE_SERVICE_NAME.yaml"

cat > "$SERVICE_RECORD_FILE" <<EOF
codename: $INPUT_SERVICE_CODENAME
repo_name: $INPUT_SERVICE_REPO_NAME
project_name: $INPUT_SERVICE_PROJECT_NAME
service_name: $INPUT_SERVICE_SERVICE_NAME
EOF

cd "$SERVICE_PATH"

# Create package.json
cat > package.json <<EOF
{
  "name": "$INPUT_SERVICE_SERVICE_NAME",
  "version": "0.1.0",
  "private": true,
  "type": "module",
  "description": "Svelte + Vite web app for $INPUT_SERVICE_PROJECT_NAME",
  "scripts": {
    "dev": "vite",
    "build": "vite build",
    "preview": "vite preview --port 3400 --host",
    "test": "vitest run",
    "check": "svelte-check --tsconfig ./tsconfig.json"
  },
  "devDependencies": {
    "@sveltejs/vite-plugin-svelte": "^7.3.1",
    "@testing-library/svelte": "^5.4.2",
    "@types/node": "^22.0.0",
    "jsdom": "^30.1.1",
    "svelte": "^5.0.0",
    "svelte-check": "^4.0.0",
    "typescript": "^5.0.0",
    "vite": "^8.0.0",
    "vitest": "^5.0.3"
  }
}
EOF

# Create vite.config.ts
cat > vite.config.ts <<EOF
import { svelte } from "@sveltejs/vite-plugin-svelte";
import { svelteTesting } from "@testing-library/svelte/vite";
import { defineConfig } from "vitest/config";
import { readFileSync } from "node:fs";

const version = (JSON.parse(
  readFileSync(new URL("./package.json", import.meta.url), "utf8")
) as { version: string }).version;

export default defineConfig({
  plugins: [svelte(), svelteTesting()],
  define: { __APP_VERSION__: JSON.stringify(version) },
  server: {
    port: 3400,
    host: true,
  },
  build: { outDir: "dist", sourcemap: false },
  test: { environment: "node", include: ["src/**/*.test.ts"] },
});
EOF

# Create tsconfig.json
cat > tsconfig.json <<EOF
{
  "compilerOptions": {
    "allowJs": true,
    "checkJs": true,
    "esModuleInterop": true,
    "forceConsistentCasingInFileNames": true,
    "resolveJsonModule": true,
    "skipLibCheck": true,
    "sourceMap": true,
    "strict": true,
    "moduleResolution": "bundler",
    "target": "ESNext",
    "module": "ESNext"
  }
}
EOF

# Create index.html
cat > index.html <<EOF
<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover" />
    <meta name="theme-color" content="#141414" />
    <meta name="color-scheme" content="dark" />
    <meta name="apple-mobile-web-app-capable" content="yes" />
    <meta name="mobile-web-app-capable" content="yes" />
    <meta name="apple-mobile-web-app-status-bar-style" content="black-translucent" />
    <meta name="apple-mobile-web-app-title" content="$INPUT_SERVICE_SERVICE_NAME" />
    <meta name="robots" content="noindex, nofollow" />
    <title>$INPUT_SERVICE_SERVICE_NAME</title>
  </head>
  <body>
    <div id="app"></div>
    <script type="module" src="/src/main.ts"></script>
  </body>
</html>
EOF

# Create src/main.ts
cat > src/main.ts <<EOF
import { mount } from "svelte";
import "./app.css";
import App from "./App.svelte";

mount(App, { target: document.getElementById("app")! });
EOF

# Create src/App.svelte
cat > src/App.svelte <<EOF
<script lang="ts">
  let count = $state(0);
</script>

<div class="app">
  <h1>Welcome to $INPUT_SERVICE_SERVICE_NAME</h1>
  <p>Count: {count}</p>
  <button onclick={() => count++}>Increment</button>
</div>

<style>
  .app {
    padding: 24px;
    max-width: 600px;
    margin: 0 auto;
  }
  h1 {
    font-size: 24px;
    margin-bottom: 16px;
  }
  button {
    padding: 8px 16px;
    border-radius: 8px;
    border: 1px solid #333;
    background: #222;
    color: white;
    cursor: pointer;
  }
  button:hover {
    background: #333;
  }
</style>
EOF

# Create src/app.css
cat > src/app.css <<EOF
* {
  box-sizing: border-box;
  margin: 0;
  padding: 0;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Oxygen, Ubuntu, Cantarell, "Open Sans", "Helvetica Neue", sans-serif;
  background: #141414;
  color: #e0e0e0;
  line-height: 1.5;
}
EOF

# Create src/App.test.ts
cat > src/App.test.ts <<EOF
import { render, screen } from "@testing-library/svelte";
import { describe, it, expect } from "vitest";
import App from "./App.svelte";

describe("App", () => {
  it("renders welcome message", () => {
    render(App);
    expect(screen.getByText(/welcome to/i)).toBeInTheDocument();
  });
});
EOF

# Create Dockerfile
cat > Dockerfile <<EOF
FROM node:22-alpine AS builder
WORKDIR /app
COPY package.json pnpm-lock.yaml* ./
RUN npm install -g pnpm && pnpm install --frozen-lockfile
COPY . .
RUN pnpm build

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx/nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
EOF

# Create nginx config
cat > nginx/nginx.conf <<EOF
server {
  listen 80;
  server_name localhost;
  root /usr/share/nginx/html;
  index index.html;

  location / {
    try_files \$uri \$uri/ /index.html;
  }

  gzip on;
  gzip_types text/plain text/css application/json application/javascript text/xml application/xml;
}
EOF

# Create .gitignore
cat > .gitignore <<EOF
node_modules
dist
.svelte-kit
.DS_Store
.env
.env.local
EOF

cd -
