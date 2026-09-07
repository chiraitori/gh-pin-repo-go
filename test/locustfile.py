import random
from locust import HttpUser, task, between, events
import urllib3

# Suppress insecure request warnings if needed
urllib3.disable_warnings()

SAMPLE_USERS = [
    "torvalds",
    "google",
    "microsoft",
    "golang",
    "facebook",
    "vuejs",
]

class GitHubPinRepoUser(HttpUser):
    """
    Simulates high-concurrency users querying the GitHub Pinned Repos API.
    Handles massive concurrent loads gracefully due to Sharded In-Memory Caching
    and Go Singleflight thundering-herd protection.
    """
    # Simulate high throughput with minimal delay
    wait_time = between(0.001, 0.05)

    @task(70)
    def test_pinned_repos_cached(self):
        """
        Simulates 70% of real-world traffic hitting frequently queried profiles
        served directly from memory cache at sub-millisecond speeds.
        """
        username = random.choice(SAMPLE_USERS)
        with self.client.get(
            f"/api/v1/pinned/{username}",
            name="/api/v1/pinned/[username]",
            catch_response=True,
        ) as response:
            if response.status_code == 200:
                data = response.json()
                if data.get("success") is True:
                    response.success()
                else:
                    response.failure(f"Unexpected response format: {response.text[:100]}")
            else:
                response.failure(f"HTTP {response.status_code}: {response.text[:100]}")

    @task(25)
    def test_health_check(self):
        """
        Simulates 25% traffic checking health or monitoring probes.
        """
        with self.client.get("/health", name="/health", catch_response=True) as response:
            if response.status_code == 200:
                response.success()
            else:
                response.failure(f"Health check failed with {response.status_code}")

    @task(4)
    def test_swagger_doc(self):
        """
        Simulates documentation exploration.
        """
        self.client.get("/swagger/doc.json", name="/swagger/doc.json")

    @task(1)
    def test_cache_refresh(self):
        """
        Simulates occasional cache-busting requests (1% traffic).
        """
        username = random.choice(SAMPLE_USERS)
        self.client.get(
            f"/api/v1/pinned/{username}?refresh=true",
            name="/api/v1/pinned/[username]?refresh=true"
        )
