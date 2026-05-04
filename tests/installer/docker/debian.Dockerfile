FROM debian:12-slim
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends \
        bash curl ca-certificates tar bats grep coreutils \
    && rm -rf /var/lib/apt/lists/*
RUN useradd -m -s /bin/bash tester
USER tester
WORKDIR /home/tester
ENV HOME=/home/tester
COPY --chown=tester:tester install.sh   /home/tester/install.sh
COPY --chown=tester:tester uninstall.sh /home/tester/uninstall.sh
COPY --chown=tester:tester tests/installer/bats /home/tester/bats
CMD ["bats", "/home/tester/bats"]
