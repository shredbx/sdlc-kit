# The image of a bos web app, assembled from a build that was made beforehand (`pnpm build` with
# adapter-node, which bundles the app and its linked framework into build/), so building the image
# needs its base image and nothing else (no package downloads, no node_modules).
#
# Build context: a folder that holds that `build/` and the app's `package.json` (its "type": "module"
# is what makes the build run as ES modules).
# The web listens on 3000 inside the container; the bundle publishes it on a host port of its own.
# PUBLIC_API_URL is where the web reaches the API on the bundle's network: the service named bos-api.
FROM node:20-alpine

WORKDIR /app
COPY package.json ./
COPY build ./build

ENV NODE_ENV=production \
    PORT=3000 \
    PUBLIC_API_URL=http://bos-api:8080
USER node
EXPOSE 3000

CMD ["node", "build"]
