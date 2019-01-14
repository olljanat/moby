def withGithubStatus(String context, Closure cl) {
    def setGithubStatus = { String state ->
        try {
            def backref = "${BUILD_URL}flowGraphTable/"
            def reposSourceURL = scm.repositories[0].getURIs()[0].toString()
            step(
                $class: 'GitHubCommitStatusSetter',
                contextSource: [$class: "ManuallyEnteredCommitContextSource", context: context],
                errorHandlers: [[$class: 'ShallowAnyErrorHandler']],
                reposSource: [$class: 'ManuallyEnteredRepositorySource', url: reposSourceURL],
                statusBackrefSource: [$class: 'ManuallyEnteredBackrefSource', backref: backref],
                statusResultSource: [$class: 'ConditionalStatusResultSource', results: [[$class: 'AnyBuildResult', state: state]]],
            )
        } catch (err) {
            echo "Exception from GitHubCommitStatusSetter for $context: $err"
        }
    }

    setGithubStatus 'PENDING'
    try {
        cl()
    } catch (err) {
        // AbortException signals a "normal" build failure.
        if (!(err instanceof hudson.AbortException)) {
            echo "Exception in withGithubStatus for $context: $err"
        }
        addFailedStep context
        setGithubStatus 'FAILURE'
        throw err
    }
    setGithubStatus 'SUCCESS'
}


pipeline {
  agent none
  options { buildDiscarder(logRotator(daysToKeepStr: '30')) }
  parameters {
        booleanParam(name: 'janky', defaultValue: true, description: 'x86 Build/Test')
        booleanParam(name: 'experimental', defaultValue: true, description: 'x86 Experimental Build/Test ')
        booleanParam(name: 'z', defaultValue: true, description: 'IBM Z (s390x) Build/Test')
        booleanParam(name: 'powerpc', defaultValue: true, description: 'PowerPC (ppc64le) Build/Test')
        booleanParam(name: 'vendor', defaultValue: true, description: 'Vendor')
        booleanParam(name: 'windowsRS1', defaultValue: true, description: 'Windows 2016 (RS1) Build/Test')
        booleanParam(name: 'windowsRS5', defaultValue: true, description: 'Windows 2019 (RS5) Build/Test')
  }
  stages {
    stage('Build') {
      parallel {
        stage('janky') {
          when {
            beforeAgent true
            expression { params.janky }
          }
          agent {
            node {
              label 'ubuntu-1604-overlay2-stable'
            }
          }
          steps {
            withCredentials([string(credentialsId: '52af932f-f13f-429e-8467-e7ff8b965cdb', variable: 'CODECOV_TOKEN')]) {
              withGithubStatus('janky') {
                sh '''
                  hack/ci/run-ci
                '''
              }
            }
          }
          post {
            always {
              sh '''
                hack/ci/stop-ci
              '''
              archiveArtifacts artifacts: 'bundles.tar.gz'
            }
          }
        }
        stage('experimental') {
          when {
            beforeAgent true
            expression { params.experimental }
          }
          agent {
            node {
              label 'ubuntu-1604-aufs-stable'
            }
          }
          steps {
            withGithubStatus('experimental') {
              sh '''
                hack/ci/run-ci experimental
              '''
            }
          }
          post {
            always {
              sh '''
                hack/ci/stop-ci experimental
              '''
              archiveArtifacts artifacts: 'bundles.tar.gz'
            }
          }
        }
        stage('z') {
          when {
            beforeAgent true
            expression { params.z }
          }
          agent {
            node {
              label 's390x-ubuntu-1604'
            }
          }
          steps {
            withGithubStatus('z') {
              sh '''
                hack/ci/run-ci s390x
              '''
            }
          }
          post {
            always {
              sh '''
                hack/ci/stop-ci s390x
              '''
              archiveArtifacts artifacts: 'bundles.tar.gz'
            }
          }
        }
        stage('powerpc') {
          when {
            beforeAgent true
            expression { params.powerpc }
          }
          agent {
            node {
              label 'ppc64le-ubuntu-1604'
            }
          }
          steps {
            withGithubStatus('powerpc') {
              sh '''
                hack/ci/run-ci ppc64le
              '''
            }
          }
          post {
            always {
              sh '''
                hack/ci/stop-ci ppc64le
              '''
              archiveArtifacts artifacts: 'bundles.tar.gz'
            }
          }
        }
        stage('vendor') {
          when {
            beforeAgent true
            expression { params.vendor }
          }
          agent {
            node {
              label 'ubuntu-1604-aufs-stable'
            }
          }
          steps {
            withGithubStatus('vendor') {
              sh '''
                hack/ci/run-ci vendor
              '''
            }
          }
          post {
            always {
              sh '''
                hack/ci/stop-ci vendor
              '''
            }
          }
        }
        stage('windowsRS1') {
          when {
            beforeAgent true
            expression { params.windowsRS1 }
          }
          agent {
            node {
              label 'windows-rs1'
            }
          }
          steps {
            withGithubStatus('windowsRS1') {
              powershell '''
                $ErrorActionPreference = 'Stop'
                .\\hack\\ci\\windows.ps1
                exit $LastExitCode
              '''
            }
          }
        }
        stage('windowsRS5-process') {
          when {
            beforeAgent true
            expression { params.windowsRS5 }
          }
          agent {
            node {
              label 'windows-rs5'
            }
          }
          steps {
            withGithubStatus('windowsRS5-process') {
              powershell '''
                $ErrorActionPreference = 'Stop'
                .\\hack\\ci\\windows.ps1
                exit $LastExitCode
              '''
            }
          }
        }
      }
    }
  }
}