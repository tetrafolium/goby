FROM node:lts-alpine AS linter-stage

### Install golang ...
RUN apk add --update --no-cache git go && \
    echo "+++ $(git version)" && \
    echo "+++ $(go version)"

ENV GOBIN="$GOROOT/bin" \
    GOPATH="/.go" \
    PATH="${GOPATH}/bin:/usr/local/go/bin:$PATH"

### Install dockerfilelint ...
RUN npm install -g dockerfilelint && \
    echo "+++ dockerfilelint $(dockerfilelint --version)"

ENV REPOPATH="github.com/tetrafolium/goby" \
    TOOLPATH="github.com/tetrafolium/inspecode-tasks"
ENV REPODIR="${GOPATH}/src/${REPOPATH}" \
    TOOLDIR="${GOPATH}/src/${TOOLPATH}"

ARG OUTDIR
ENV OUTDIR="${OUTDIR:-"/.reports"}"

RUN mkdir -p "${REPODIR}" "${OUTDIR}"
COPY . "${REPODIR}"
WORKDIR "${REPODIR}"

### Put symlink refers submodule-path at origin-path
RUN ln -s "${REPODIR}/$(basename "${TOOLPATH}")" "${TOOLDIR}"

RUN ( ls -ld "${REPODIR}/inspecode-tasks/" && ls -la "${REPODIR}/inspecode-tasks/" ) || true
RUN ( ls -ld "${TOOLDIR}/" && ls -lLa "${TOOLDIR}/" ) || true

### Run dockerfilelint ...
SHELL ["/bin/ash", "-eo", "pipefail", "-c"]
RUN ( find . -type f -name '*Dockerfile*' -print0 | xargs -0 dockerfilelint --output json ) \
        > "${OUTDIR}/dockerfilelint.json" || true
RUN ls -la "${OUTDIR}"

RUN ( ls -ld "${REPODIR}/inspecode-tasks/" && ls -la "${REPODIR}/inspecode-tasks/" ) || true
RUN ( ls -ld "${TOOLDIR}/" && ls -lLa "${TOOLDIR}/" ) || true

### Convert dockerfilelint JSON to SARIF ...
RUN go run "${TOOLDIR}/dockerfilelint/cmd/main.go" "${REPOPATH}" \
        < "${OUTDIR}/dockerfilelint.json" \
        > "${OUTDIR}/dockerfilelint.sarif"
RUN ls -la "${OUTDIR}"
