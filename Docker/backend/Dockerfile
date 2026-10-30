# 백엔드 개발용 이미지. 소스는 compose에서 /app 으로 바인드 마운트한다.
FROM golang:1.27-alpine3.24

RUN apk add --no-cache git tzdata

WORKDIR /app

COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN sed -i 's/\r$//' /usr/local/bin/entrypoint.sh && chmod +x /usr/local/bin/entrypoint.sh

EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
