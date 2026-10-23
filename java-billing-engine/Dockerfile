FROM maven:3.9.9-eclipse-temurin-17 AS build
WORKDIR /src
COPY pom.xml .
COPY src ./src
RUN mvn -B -DskipTests package

FROM eclipse-temurin:17-jre-jammy
WORKDIR /opt/app
RUN apt-get update \
    && apt-get install -y --no-install-recommends wget \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --uid 65532 --home-dir /opt/app --shell /usr/sbin/nologin app
COPY --from=build /src/target/billing-engine-0.1.0.jar /opt/app/app.jar
USER 65532
EXPOSE 8080
ENTRYPOINT ["java", "-jar", "/opt/app/app.jar"]
