pluginManagement {
    repositories {
        mavenLocal()
        gradlePluginPortal()
        maven("https://dl.bintray.com/hypertrace/maven")
    }
}

plugins {
    id("org.hypertrace.version-settings") version "0.1.0"
}

rootProject.name = "kafka-topic-creator"
