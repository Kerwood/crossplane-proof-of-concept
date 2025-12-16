FROM node:22-alpine

WORKDIR /app
COPY package.json server.js ./
RUN npm install
EXPOSE 3000

CMD ["server.js"]
