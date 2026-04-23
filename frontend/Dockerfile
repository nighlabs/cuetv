FROM node:25-alpine AS builder

WORKDIR /app
COPY package.json package-lock.json ./
# npm install instead of npm ci: the lockfile is generated on macOS ARM
# and @emnapi optional native deps resolve differently on Linux x64.
RUN npm install
COPY . .
RUN npm run build

FROM nginx:alpine

COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO- http://localhost:80/ || exit 1
