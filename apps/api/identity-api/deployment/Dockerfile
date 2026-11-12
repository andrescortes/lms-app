FROM eclipse-temurin:21-jdk-alpine
# NOTE: If the Java base image already provides the user and its permissions, remove this line
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
WORKDIR /app
COPY *.jar identity-api.jar
# NOTE: If the Java base image already provides the user and its permissions, remove this line
RUN chown -R appuser:appgroup /app
ENV JAVA_OPTS="-XX:MaxRAMPercentage=70 -XX:+ExitOnOutOfMemoryError -XX:+HeapDumpOnOutOfMemoryError -Djava.security.egd=file:/dev/./urandom"
# Replace with a non-root user to avoid running the container with excessive privileges
USER appuser
ENTRYPOINT [ "/bin/sh", "-c", "/opt/java/openjdk/bin/java $JAVA_OPTS  -jar /app/identity-api.jar" ]
