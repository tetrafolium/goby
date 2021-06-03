FROM python:3-alpine AS linter-stage

### Install golang ...
RUN apk add --update --no-cache go && \
    echo "+++ $(go version)"

ENV GOBIN="$GOROOT/bin" \
    GOPATH="/.go" \
    PATH="${GOPATH}/bin:/usr/local/go/bin:$PATH"

### Install yamllint tool ...
RUN pip3 install 'yamllint>=1.24.0,<1.25.0' && \
    echo "+++ $(yamllint --version)"

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

### Run yamllint ...
RUN yamllint -f parsable . > "${OUTDIR}/yamllint.issues" || true
RUN ls -la "${OUTDIR}"

RUN ( ls -ld "${REPODIR}/inspecode-tasks/" && ls -la "${REPODIR}/inspecode-tasks/" ) || true
RUN ( ls -ld "${TOOLDIR}/" && ls -lLa "${TOOLDIR}/" ) || true

### Convert yamllint issues to SARIF ...
RUN go run "${TOOLDIR}/yamllint/cmd/main.go" "${REPOPATH}" \
        < "${OUTDIR}/yamllint.issues" \
        > "${OUTDIR}/yamllint.sarif"
RUN ls -la "${OUTDIR}"
