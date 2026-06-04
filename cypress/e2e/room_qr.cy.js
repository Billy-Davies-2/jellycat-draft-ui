describe("room join QR and public draft", () => {
  it("publishes room metadata and serves a PNG QR code", () => {
    cy.request("/api/room").then(({ body }) => {
      expect(body.code).to.match(/^[A-Z0-9]{4}$/);
      expect(body.joinPath).to.eq(`/join?code=${body.code}`);
      expect(body.joinUrl).to.include(`/join?code=${body.code}`);
      expect(body.qrPath).to.eq(`/api/room/qr?code=${body.code}`);

      cy.request({ url: body.qrPath, encoding: "binary" }).then((response) => {
        expect(response.status).to.eq(200);
        expect(response.headers["content-type"]).to.include("image/png");
        expect(response.body.charCodeAt(0)).to.eq(0x89);
        expect(response.body.slice(1, 4)).to.eq("PNG");
      });
    });
  });

  it("serves Jellycat assets through the app image route", () => {
    cy.request({ url: "/images/bashful-bunny.png", encoding: "binary" }).then((response) => {
      expect(response.status).to.eq(200);
      expect(response.headers["content-type"]).to.include("image/png");
      expect(response.body.charCodeAt(0)).to.eq(0x89);
      expect(response.body.slice(1, 4)).to.eq("PNG");
    });
  });

  it("shows anonymous Big Board with a scannable QR but no admin identity", () => {
    cy.visit("/draft");

    cy.contains("Join From Phone").should("be.visible");
    cy.get("img.room-qr")
      .should("be.visible")
      .and("have.attr", "src")
      .and("match", /^\/api\/room\/qr\?code=[A-Z0-9]{4}$/);
    cy.get("img[src^='/images/']").should("have.length.greaterThan", 0);
    cy.get("img[src^='/static/images/']").should("not.exist");

    cy.contains("Welcome, ").should("not.exist");
    cy.contains("Billy").should("not.exist");
    cy.contains("Admin Panel").should("not.exist");
    cy.contains("Set Draft Style").should("not.exist");
  });

  it("keeps explicit dev login available for commissioner controls", () => {
    cy.visit("/auth/login");
    cy.location("pathname").should("eq", "/start");

    cy.visit("/draft");
    cy.contains("Welcome,").should("be.visible");
    cy.contains("Billy").should("be.visible");
    cy.contains("Admin Panel").should("be.visible");
    cy.contains("Set Draft Style").should("be.visible");
  });
});