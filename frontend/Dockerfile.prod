# Build stage
FROM node:22-alpine AS builder

WORKDIR /app

# Copy package.json and lock file
COPY frontend/package.json frontend/yarn.lock ./
RUN yarn install

# Copy the rest of the source code
COPY frontend/ ./

# Build the application
RUN yarn build

# Final stage
FROM nginx:stable-alpine

COPY frontend/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/dist /usr/share/nginx/html

# Expose the frontend port
EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
