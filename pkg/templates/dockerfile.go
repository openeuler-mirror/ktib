package templates

// Dockerfile is a default Dockerfile laid down by s2i create
const Dockerfile = `# {{.ImageName}}
FROM BaseImage

# TODO: Put the maintainer name in the image metadata
# LABEL maintainer="Your Name <your@email.com>"

# TODO: Rename the builder environment variable to inform users about application you provide them
# ENV BUILDER_VERSION 1.0

# TODO: Set labels to describe the builder image
#LABEL io.k8s.description="Platform for building xyz" \
#      io.k8s.display-name="builder x.y.z" 

# TODO: Install required packages here:
# RUN yum install -y ... && yum clean all -y
RUN yum install -y rubygems && yum clean all -y
RUN gem install asdf

# TODO (optional): Copy the builder files into /opt/app-root
# COPY ./<builder_folder>/ /opt/app-root/
COPY ./init/bin/xxx /usr/bin/xxx

# This default user is created in the base image
USER 1001

# TODO: Set the default port for applications built using this image
# EXPOSE 8080

# TODO: Set the default CMD for the image
# CMD ["/usr/bin/xxx"]
`
