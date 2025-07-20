# Use Node.js 18 LTS as base image
FROM node:22-alpine

# Set working directory
WORKDIR /app

# Copy package files
COPY package*.json ./

RUN npm i -g tsx typescript

# Install dependencies
RUN npm install


# Copy source code
COPY . .

# Build the TypeScript code
RUN npm run build

# Expose the P2P port (9901 by default)
EXPOSE 9901

# Set environment variables for better container compatibility
ENV NODE_ENV=production

# Run the demo
CMD ["npm", "run", "demo"]
