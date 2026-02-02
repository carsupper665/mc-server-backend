from __future__ import annotations

import json
from typing import Any, Dict, List, Optional, Sequence

import requests


MODRINTH_API = "https://api.modrinth.com/v2"
HEADERS = {
    "Accept": "application/json",
    "User-Agent": "bless/modrinth-tools/0.1",
}

def modrinth_search_mods(
    query: str,
    *,
    limit: int = 10,
    offset: int = 0,
    loader: Optional[str] = None,          # e.g. "fabric", "forge", "quilt", "neoforge"
    game_version: Optional[str] = None,    # e.g. "1.20.1"
    categories_any: Optional[Sequence[str]] = None,  # OR: any of these categories
    index: str = "relevance",              # relevance|downloads|follows|newest|updated
    timeout: float = 30.0,
) -> Dict[str, Any]:
    """
    Search Modrinth projects, constrained to project_type=mod.

    Facets logic (per Modrinth):
      - Same inner array => OR
      - Different inner arrays => AND
    """
    facet_groups: List[List[str]] = [["project_type:mod"]]

    # loaders are lumped into `categories` facets in search
    if loader:
        facet_groups.append([f"categories:{loader}"])
    if game_version:
        facet_groups.append([f"versions:{game_version}"])
    if categories_any:
        facet_groups.append([f"categories:{c}" for c in categories_any])  # OR within this group

    params = {
        "query": query,
        "limit": max(1, min(int(limit), 100)),  # keep it sane
        "offset": max(0, int(offset)),
        "index": index,
        "facets": json.dumps(facet_groups),
    }

    headers = {
        "Accept": "application/json",
        "User-Agent": "Bless-Modrinth-Search/1.0",
    }

    url = f"{MODRINTH_API}/search"

    print(url)

    r = requests.get(url, params=params, headers=headers, timeout=timeout)
    r.raise_for_status()
    return r.json()


def modrinth_get_projects(project_ids: List[str], *, timeout: float = 15.0) -> List[Dict[str, Any]]:
    """
    Bulk get projects: GET /projects?ids=[...]
    Response includes `versions`: list of version IDs.
    """
    r = requests.get(
        f"{MODRINTH_API}/projects",
        params={"ids": json.dumps(project_ids)},
        headers=HEADERS,
        timeout=timeout,
    )
    r.raise_for_status()
    return r.json()

def modrinth_pick_fields_with_version_ids(search_json: Dict[str, Any], *, timeout: float = 15.0) -> List[Dict[str, Any]]:
    hits = search_json.get("hits", []) or []
    project_ids = [h.get("project_id") for h in hits if h.get("project_id")]

    # 批次把每個 project 的 versions(version IDs) 拿回來
    projects = modrinth_get_projects(project_ids, timeout=timeout)
    versions_by_project_id = {p.get("id"): (p.get("versions") or []) for p in projects}

    out: List[Dict[str, Any]] = []
    for h in hits:
        pid = h.get("project_id")
        out.append({
            "project_id": pid,
            "slug": h.get("slug"),
            "title": h.get("title"),
            "version_ids": versions_by_project_id.get(pid, []),
        })
    return out

if __name__ == "__main__":
    data = modrinth_search_mods(
        "sodium",
        limit=10,
        loader="fabric",
        game_version="1.20.1",
        categories_any=["performance", "optimization"],
        index="downloads",
    )
    for row in modrinth_pick_fields_with_version_ids(data):
        print(row["project_id"], row["slug"], row["title"], "-", row["version_ids"][0])
