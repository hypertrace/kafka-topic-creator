pluginManagement {
  repositories {
    mavenLocal()
    gradlePluginPortal()
    maven("https://us-maven.pkg.dev/hypertrace-repos/maven")
  }
}

plugins {
  id("org.hypertrace.version-settings") version "0.3.0"
}

rootProject.name = "kafka-topic-creator"
