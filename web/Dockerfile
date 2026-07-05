FROM node:24-alpine AS build
RUN npm install --global elm@0.19.1-6
WORKDIR /src
COPY package.json package-lock.json elm.json ./
RUN npm ci
COPY src ./src
COPY public ./public
RUN npm run build

FROM nginx:1.29-alpine
COPY --from=build /src/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 8080
