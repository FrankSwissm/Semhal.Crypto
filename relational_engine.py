class RelationalMap:
    def __init__(self):
        self.lineages = {
            "Shem": ["Oromo", "Somali", "Afar", "Beja", "Amara"],
            "Ham": ["Agew"],
            "Yam": ["Sidama"]
        }
        self.anchor = "Fixed Point"

    def get_group_data(self, ethnic_group):
        for branch, groups in self.lineages.items():
            if ethnic_group in groups:
                return {"lineage": branch, "status": "Verified"}
        return {"lineage": "Unknown", "status": "Pending"}
