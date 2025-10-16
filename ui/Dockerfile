# syntax=docker/dockerfile:1
FROM node:20-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json* yarn.lock* pnpm-lock.yaml* .npmrc* ./
RUN if [ -f package-lock.json ]; then npm ci; elif [ -f yarn.lock ]; then yarn --frozen-lockfile; elif [ -f pnpm-lock.yaml ]; then npm i -g pnpm && pnpm i --frozen-lockfile; else npm i; fi

FROM node:20-alpine AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
# With output: 'export', next build emits static site into /app/out
RUN npm run build

FROM nginx:1.27-alpine AS runtime
COPY --from=builder /app/out /usr/share/nginx/html/app
EXPOSE 80
CMD ["nginx","-g","daemon off;"]
