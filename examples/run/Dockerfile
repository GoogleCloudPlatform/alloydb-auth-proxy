# Copyright 2026 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM node:24-slim AS builder
COPY . /app
WORKDIR /app

RUN npm ci
RUN npm run build

FROM gcr.io/distroless/nodejs24-debian13 AS runtime
COPY --from=builder /app /app
WORKDIR /app

ENV NODE_ENV=production
EXPOSE 8080
CMD ["dist/main.js"]
