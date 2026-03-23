FROM node:22-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
ARG VITE_BFF_URL
ENV VITE_BFF_URL=$VITE_BFF_URL
RUN npm run build

FROM caddy:2-alpine
COPY --from=builder /app/dist /srv
COPY Caddyfile /etc/caddy/Caddyfile
