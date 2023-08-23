#
#  docker run -p 8080:8080 -p 9000:9000 tempo-serverless
#
#  to exercise the function after building using the raw lambda interface
#    curl -d '{}' http://localhost:8080/2015-03-31/functions/function/invocations
#    passed body should be serializable to: events.ALBTargetGroupRequest
#    response body will be: events.ALBTargetGroupResponse
#
#  to exercise the function as if it were an http endpoint through an alb
#    curl http://localhost:9000/?start=1000&end=1001&...
#

#
# build the lambda and retrive the lambda-local-proxy
#
FROM golang:1.21.0 AS build

# copy in the lambda. todo: build in container
COPY lambda /

# copy in https://github.com/treasure-data/lambda-local-proxy to include in the docker image.
#   this project acts as an "alb" and translates normal http requests into Lambda invokes
RUN    curl -L https://github.com/treasure-data/lambda-local-proxy/releases/download/v0.0.5/lambda-local-proxy_0.0.5_Linux_x86_64.tar.gz | tar xvz -C / \
  && chmod +x /lambda-local-proxy

#
# https://docs.aws.amazon.com/lambda/latest/dg/images-create.html
#
FROM public.ecr.aws/lambda/go:1.2022.01.25.13

COPY --from=build /lambda ${LAMBDA_TASK_ROOT}
COPY --from=build /lambda-local-proxy /lambda-local-proxy

RUN chmod +x /lambda-local-proxy

RUN echo -e '\
  #!/bin/bash \n\
  AWS_REGION=dummy AWS_ACCESS_KEY_ID=dummy AWS_SECRET_ACCESS_KEY=dummy /lambda-local-proxy -e http://localhost:8080 -p 9000 -f function & \n\
  exec /lambda-entrypoint.sh lambda \n\
  ' > /run.sh
RUN chmod +x /run.sh

# run lambda-local-proxy and the actual lambda runtime itself.
ENTRYPOINT /run.sh
