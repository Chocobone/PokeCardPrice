# 프론트엔드 개발용 이미지. 소스는 compose에서 /app 으로 바인드 마운트한다.
FROM node:26-alpine3.24

RUN apk add --no-cache tzdata

WORKDIR /app

COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN sed -i 's/\r$//' /usr/local/bin/entrypoint.sh && chmod +x /usr/local/bin/entrypoint.sh

EXPOSE 5173
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
