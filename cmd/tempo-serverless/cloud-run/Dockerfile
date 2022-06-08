# 
#  docker run -p 8080:8080 tempo-serverless
#
#  to exercise the function
#    curl http://localhost:8080/?start=1000&end=1001&...
#

#
# build the serverless container image
#
FROM alpine:3.15 as certs
RUN apk --update add ca-certificates
COPY ./main /main
ENTRYPOINT ["/main"]