FROM phusion/baseimage:0.9.17
RUN apt-get update && apt-get install -y git-core curl
COPY ./phabricator-circleci /phabricator-circleci
CMD ["/phabricator-circleci"]

RUN apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*
