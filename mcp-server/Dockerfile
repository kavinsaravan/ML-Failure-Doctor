# Dependencies stage
FROM node:18-alpine AS deps

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm ci --production

# Runner stage
FROM node:18-alpine AS runner

WORKDIR /app

ENV NODE_ENV production

# Create non-root user
RUN addgroup --system --gid 1001 mcpuser
RUN adduser --system --uid 1001 mcpuser

# Copy dependencies and source
COPY --from=deps /app/node_modules ./node_modules
COPY server.js .
COPY package*.json ./

USER mcpuser

# MCP servers typically use stdio, but we can expose a port if needed
EXPOSE 3001

CMD ["node", "server.js"]
