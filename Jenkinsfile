#!groovy

// Pipeline documentation: https://jenkins.io/doc/pipeline/
// Groovy syntax reference: http://groovy-lang.org/syntax.html

// Node executes on 64bit linux only
//node('unix && 64bit') {
node {

  //def err = null
  //currentBuild.result = FAILURE

  // try {

    // no longer needed if node ('linux && 64bit') was used...
    if (!isUnix()) {
        error "This file can only run on unix-like systems."
    }

    def PACKAGE_NAME = 'github.com/almighty/almighty-core'

    stage 'Checkout from SCM'

      def checkoutDir = "go/src/${PACKAGE_NAME}"
      sh "mkdir -pv ${checkoutDir}"

      dir ("${checkoutDir}") {
        checkout scm
      }

    def v = version()
    echo "Version is ${v}"

    stage 'Create builder image'

      def builderImageTag = "almighty-core-builder-image:" + env.BRANCH_NAME + "-" + env.BUILD_NUMBER
      // Path to where to find the builder's "Dockerfile"
      def builderImageDir = "jenkins/docker/builder"
      def builderImage = docker.build(builderImageTag, builderImageDir)

    stage 'Build with builder container'

      builderImage.withRun {
        // Setup GOPATH
        def currentDir = pwd()
        def GOPATH = "${currentDir}/go"
        def PACKAGE_PATH = "${GOPATH}/src/${PACKAGE_NAME}"
        sh "mkdir -pv ${PACKAGE_PATH}"
        sh "mkdir -pv ${GOPATH}/bin"
        sh "mkdir -pv ${GOPATH}/pkg"

        sh 'cat /etc/redhat-release'
        sh 'go version'
        sh 'git --version'
        sh 'hg --version'
        sh 'glide --version'

        dir ("${PACKAGE_PATH}") {

          env.GOPATH = "${GOPATH}"

          stage "fetch dependencies"
          sh 'make deps'

          stage "generate code"
          sh 'make generate'

          stage "build"
          sh 'make build'

          stage "unit tests"
          sh 'make test-unit'
        }

        // Add stage inside withRun {} and add a cleanup stage?
      }

    //currentBuild.result = "SUCCESS"

  //} catch (e) {

  //  def w = new StringWriter()
  //  err.printStackTrace(new PrintWriter(w))

  //  mail body: "project build error: ${err}" ,
  //  from: 'admin@your-jenkins.com',
  //  replyTo: 'noreply@your-jenkins.com',
  //  subject: 'project build failed',
  //  to: 'kkleine@redhat.com'

  //  throw err
  //}
}

def version() {
  sh 'git describe --tags --long > git-describe.out'
  def vers = readFile('commandResult').trim()
}

// Don't use "input" within a "node"
// When you use inputs, it is a best practice to wrap them in timeouts. Wrapping inputs in timeouts allows them to be cleaned up if
// approvals do not occur within a given window. For example:
//
// timeout(time:5, unit:'DAYS') {
//     input message:'Approve deployment?', submitter: 'it-ops'
// }

// Try catch blocks:
//
//     try {
//         sh 'might fail'
//         mail subject: 'all well', to: 'admin@somewhere', body: 'All well.'
//     } catch (e) {
//         def w = new StringWriter()
//         e.printStackTrace(new PrintWriter(w))
//         mail subject: "failed with ${e.message}", to: 'admin@somewhere', body: "Failed: ${w}"
//         throw e
//     }

// For headless GUI tests see https://github.com/jenkinsci/workflow-basic-steps-plugin/blob/master/CORE-STEPS.md#build-wrappers
