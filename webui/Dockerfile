# Build stage
FROM node:20-alpine AS builder

WORKDIR /app

# Copy package files
COPY package.json yarn.lock .yarnrc.yml ./
COPY .yarn/ ./.yarn/

# Install dependencies
RUN corepack enable && yarn install --immutable

# Copy source code
COPY public/ ./public/
COPY src/ ./src/
COPY index.html vite.config.js ./

# Build for production
RUN yarn build-prod

# Production stage with nginx
FROM nginx:alpine

# Copy built assets from builder
COPY --from=builder /app/dist /usr/share/nginx/html

# Copy nginx configuration
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
