FROM phusion/baseimage:0.9.17
RUN apt-get update && apt-get install -y git-core curl
COPY ./buildtrigger /buildtrigger
CMD ["/buildtrigger"]

RUN apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*
