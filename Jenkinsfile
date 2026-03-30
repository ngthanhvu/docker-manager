pipeline {
  agent any

  options {
    timestamps()
    ansiColor('xterm')
    disableConcurrentBuilds()
  }

  parameters {
    string(name: 'DOCKERHUB_NAMESPACE', defaultValue: 'ngthanhvu', description: 'Docker Hub username / namespace')
    string(name: 'REPO_PREFIX', defaultValue: 'docker-manager', description: 'Repository prefix')
    choice(name: 'SERVICE', choices: ['all', 'backend', 'frontend'], description: 'Which service to build and push')
    string(name: 'IMAGE_TAG', defaultValue: '', description: 'Image tag. Leave empty to use Git tag, then branch-buildNumber fallback')
    booleanParam(name: 'AUTO_INCREMENT_TAG', defaultValue: false, description: 'Auto-increment tag from Docker Hub using vX.Y format')
  }

  environment {
    DOCKER_BUILDKIT = '1'
    COMPOSE_DOCKER_CLI_BUILD = '1'
  }

  stages {
    stage('Checkout') {
      steps {
        checkout scm
        sh 'git submodule update --init --recursive || true'
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
            def repoToCheck = params.SERVICE == 'backend'
              ? "${params.DOCKERHUB_NAMESPACE}/${params.REPO_PREFIX}-backend"
              : "${params.DOCKERHUB_NAMESPACE}/${params.REPO_PREFIX}-frontend"

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

            if (latestTag) {
              def matcher = (latestTag =~ /^v(\\d+)\\.(\\d+)$/)
              if (!matcher.matches()) {
                error("Latest Docker Hub tag '${latestTag}' in ${repoToCheck} is not in vX.Y format")
              }

              int major = matcher[0][1] as int
              int minor = matcher[0][2] as int
              minor += 1
              if (minor >= 10) {
                major += 1
                minor = 0
              }
              resolvedTag = "v${major}.${minor}"
            } else {
              resolvedTag = 'v1.0'
            }
          }

          env.BUILD_IMAGE_TAG = resolvedTag
          env.BUILD_DATE = sh(script: 'date -u +%F', returnStdout: true).trim()
        }

        sh '''
          echo "Namespace: ${DOCKERHUB_NAMESPACE}"
          echo "Repo prefix: ${REPO_PREFIX}"
          echo "Service: ${SERVICE}"
          echo "Image tag: ${BUILD_IMAGE_TAG}"
          echo "Build date: ${BUILD_DATE}"
        '''
      }
    }

    stage('Docker Login') {
      steps {
        withCredentials([usernamePassword(
          credentialsId: 'dockerhub',
          usernameVariable: 'DOCKERHUB_USER',
          passwordVariable: 'DOCKERHUB_TOKEN'
        )]) {
          sh '''
            echo "$DOCKERHUB_TOKEN" | docker login -u "$DOCKERHUB_USER" --password-stdin
          '''
        }
      }
    }

    stage('Build And Push') {
      steps {
        sh '''
          chmod +x ./run-prod.sh
          APP_VERSION="${BUILD_IMAGE_TAG}" BUILD_DATE="${BUILD_DATE}" \
            ./run-prod.sh push "${DOCKERHUB_NAMESPACE}" "${REPO_PREFIX}" "${SERVICE}" "${BUILD_IMAGE_TAG}"
        '''
      }
    }
  }

  post {
    success {
      echo "Pushed ${params.SERVICE} image(s) to ${params.DOCKERHUB_NAMESPACE}/${params.REPO_PREFIX} with tag ${env.BUILD_IMAGE_TAG}"
    }

    always {
      sh 'docker logout || true'
      cleanWs(deleteDirs: true, notFailBuild: true)
    }
  }
}
