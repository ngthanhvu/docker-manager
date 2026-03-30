pipeline {
  agent any

  options {
    timestamps()
    disableConcurrentBuilds()
  }

  parameters {
    string(name: 'DOCKERHUB_NAMESPACE', defaultValue: 'ngthanhvu', description: 'Docker Hub namespace')
    string(name: 'REPO_PREFIX', defaultValue: 'docker-manager', description: 'Repository prefix')
    choice(name: 'SERVICE', choices: ['all', 'backend', 'frontend'], description: 'Service to build')
    string(name: 'IMAGE_TAG', defaultValue: '', description: 'Custom image tag')
    booleanParam(name: 'AUTO_INCREMENT_TAG', defaultValue: false, description: 'Auto increment vX.Y')
  }

  environment {
    DOCKER_BUILDKIT = '1'
    COMPOSE_DOCKER_CLI_BUILD = '1'
  }

  stages {

    stage('Checkout') {
      steps {
        checkout scm
        sh '''
          git config --global url."git@github.com:".insteadOf "https://github.com/" || true
          git submodule update --init --recursive || true
        '''
      }
    }

    stage('Resolve Metadata') {
      steps {
        script {
          def gitTag = env.TAG_NAME?.trim()
          def branchName = env.BRANCH_NAME?.trim() ?: 'manual'
          def safeBranch = branchName.replaceAll(/[^A-Za-z0-9._-]+/, '-')
          def requestedTag = params.IMAGE_TAG?.trim()

          def resolvedTag = requestedTag ?: (gitTag ?: "${safeBranch}-${env.BUILD_NUMBER}")

          if (params.AUTO_INCREMENT_TAG) {

            def repo = "${params.DOCKERHUB_NAMESPACE}/${params.REPO_PREFIX}-${params.SERVICE == 'backend' ? 'backend' : 'frontend'}"

            def latestTag = sh(
              script: """
                set -e
                curl -fsSL "https://hub.docker.com/v2/namespaces/${params.DOCKERHUB_NAMESPACE}/repositories/${params.REPO_PREFIX}-${params.SERVICE == 'backend' ? 'backend' : 'frontend'}/tags?page_size=100" \
                | grep -oE '"name"[[:space:]]*:[[:space:]]*"v[0-9]+\\.[0-9]+"' \
                | sed -E 's/.*"name"[[:space:]]*:[[:space:]]*"(v[0-9]+\\.[0-9]+)".*/\\1/' \
                | sort -V \
                | tail -n 1
              """,
              returnStdout: true
            ).trim()

            echo "Latest tag from DockerHub: ${latestTag}"

            if (latestTag && latestTag ==~ /^v\\d+\\.\\d+$/) {

              def parts = latestTag.replace("v", "").split("\\.")
              int major = parts[0] as int
              int minor = parts[1] as int

              minor += 1
              if (minor >= 10) {
                major += 1
                minor = 0
              }

              resolvedTag = "v${major}.${minor}"

            } else {
              echo "No valid tag found, fallback to v1.0"
              resolvedTag = 'v1.0'
            }
          }

          env.BUILD_IMAGE_TAG = resolvedTag
          env.BUILD_DATE = sh(script: 'date -u +%F', returnStdout: true).trim()
        }

        sh """
          echo "Namespace: ${params.DOCKERHUB_NAMESPACE}"
          echo "Repo prefix: ${params.REPO_PREFIX}"
          echo "Service: ${params.SERVICE}"
          echo "Image tag: ${env.BUILD_IMAGE_TAG}"
          echo "Build date: ${env.BUILD_DATE}"
        """
      }
    }

    stage('Docker Login') {
      steps {
        withCredentials([usernamePassword(
          credentialsId: 'dockerhub',
          usernameVariable: 'DOCKERHUB_USER',
          passwordVariable: 'DOCKERHUB_TOKEN'
        )]) {
          retry(3) {
            sh '''
              echo "$DOCKERHUB_TOKEN" | docker login -u "$DOCKERHUB_USER" --password-stdin
            '''
          }
        }
      }
    }

    stage('Build And Push') {
      steps {
        wrap([$class: 'AnsiColorBuildWrapper', colorMapName: 'xterm']) {
          sh '''
            chmod +x ./run-prod.sh
            APP_VERSION="${BUILD_IMAGE_TAG}" BUILD_DATE="${BUILD_DATE}" \
              ./run-prod.sh push "${DOCKERHUB_NAMESPACE}" "${REPO_PREFIX}" "${SERVICE}" "${BUILD_IMAGE_TAG}"
          '''
        }
      }
    }
  }

  post {
    success {
      echo "Pushed ${params.SERVICE} → ${params.DOCKERHUB_NAMESPACE}/${params.REPO_PREFIX}:${env.BUILD_IMAGE_TAG}"
    }

    always {
      sh 'docker logout || true'
      cleanWs(deleteDirs: true, notFailBuild: true)
    }
  }
}