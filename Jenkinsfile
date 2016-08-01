#!groovy

// Pipeline documentation: https://jenkins.io/doc/pipeline/
// Groovy syntax reference: http://groovy-lang.org/syntax.html

// Only keep the 10 most recent builds
properties([
  [ $class: 'BuildDiscarderProperty',
    strategy: [ $class: 'LogRotator', numToKeepStr: '10'] ]
])

try {
  def PACKAGE_NAME = 'github.com/almighty/almighty-core'

  // Node executes on 64bit linux only
  // node('unix && 64bit') {
  node {
    // no longer needed if node ('linux && 64bit') was used...
    if (!isUnix()) {
      error "This file can only run on unix-like systems."
    }

    stage 'Checkout project from SCM'
    def checkoutDir = "go/src/${PACKAGE_NAME}"
    sh "mkdir -pv ${checkoutDir}"
    dir ("${checkoutDir}") {
      checkout scm
    }

    // TODO: (kwk) determine version
    def v = version()
    echo "Version is ${v}"

    stage 'Create docker builder image'
    def builderImageTag = "almighty-core-builder-image:" + env.BRANCH_NAME + "-" + env.BUILD_NUMBER
    // Path to where to find the builder's "Dockerfile"
    def builderImageDir = "jenkins/docker/builder"
    def builderImage = docker.build(builderImageTag, builderImageDir)

    builderImage.withRun {c ->
      // Setup GOPATH
      def currentDir = pwd()
      def GOPATH = "${currentDir}/go"
      def PACKAGE_PATH = "${GOPATH}/src/${PACKAGE_NAME}"

      dir ("${PACKAGE_PATH}") {
        env.GOPATH = "${GOPATH}"
        stage "Fetch Go package dependencies"
        sh 'make deps'
        stage "Generate controllers from Goa design code"
        sh 'make generate'
        stage "Go build"
        sh 'make build'
        stage "Run unit tests"
        sh 'make test-unit'
        stage "Run integration tests"
        sh 'make test-integration'
        // TODO: (kwk) a cleanup stage?
      }

      sh "docker logs ${c.id}"
    }
  } // end of node {}
} catch (exc) {
  echo "An error occured. Handling it now."

  def w = new StringWriter()
  exc.printStackTrace(new PrintWriter(w))

  emailext subject: "${env.JOB_NAME} (${env.BUILD_NUMBER}) failed",
    body: "It appears that ${env.BUILD_URL} is failing, somebody should do something about that",
    to: 'kkleine@redhat.com',
    recipientProviders: [
      // Sends email to all the people who caused a change in the change set:
      [$class: 'DevelopersRecipientProvider'],
      // Sends email to the user who initiated the build:
      [$class: 'RequesterRecipientProvider']
    ],
    replyTo: 'noreply@localhost',
    attachLog: true

  throw err
}

def version() {
  //sh 'git describe --tags --long > git-describe.out'
  //def vers = readFile('commandResult').trim()
  def vers = "v0.0.1"
}

// Don't use "input" within a "node"
// When you use inputs, it is a best practice to wrap them in timeouts. Wrapping inputs in timeouts allows them to be cleaned up if
// approvals do not occur within a given window. For example:
//
// timeout(time:5, unit:'DAYS') {
//     input message:'Approve deployment?', submitter: 'it-ops'
// }

// For headless GUI tests see https://github.com/jenkinsci/workflow-basic-steps-plugin/blob/master/CORE-STEPS.md#build-wrappers
