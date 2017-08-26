FROM elixir

ENV TERM=xterm

RUN apt-get update -y && \
    apt-get install -y inotify-tools && \
    apt-get clean && rm -rf /var/lib/apt/lists/*


ADD https://raw.githubusercontent.com/vishnubob/wait-for-it/master/wait-for-it.sh /bin/wait-for-it.sh
RUN chmod +x /bin/wait-for-it.sh

RUN mix local.hex --force && \
    mix local.rebar --force

RUN mkdir /app

WORKDIR /app

VOLUME /app
