describe("room join QR and public draft", () => {
  function pairPhone(room, username, teamName) {
    return cy
      .request("POST", "/api/room/join", {
        code: room.code,
        username,
        teamName,
      })
      .then(({ body }) => body.team);
  }

  function pairExistingPhone(room, username, team) {
    return cy
      .request("POST", "/api/room/join", {
        code: room.code,
        username,
        teamId: team.id,
      })
      .then(({ body }) => body.team);
  }

  function visitPairedPhone(room, team, username) {
    cy.visit(`/pick?code=${room.code}`, {
      onBeforeLoad(win) {
        win.localStorage.setItem("jellycat_room_code", room.code);
        win.localStorage.setItem("jellycat_owner", username);
        win.localStorage.setItem("jellycat_team_id", team.id);
        win.localStorage.setItem("jellycat_team_name", team.name);
      },
    });
  }

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

  it("renders the QR code image on the Room Display", () => {
    cy.visit("/draft");

    cy.get("img.room-qr")
      .should("be.visible")
      .and(($image) => {
        const image = $image[0];
        expect(image.complete).to.eq(true);
        expect(image.naturalWidth).to.be.greaterThan(100);
        expect(image.naturalHeight).to.be.greaterThan(100);
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

  it("requires room-code proof before accepting controller picks", () => {
    cy.request("/api/draft/state").then(({ body }) => {
      const player = body.players.find((candidate) => !candidate.drafted);

      cy.request({
        method: "POST",
        url: "/api/draft/pick",
        failOnStatusCode: false,
        body: { playerId: player.id, teamId: body.currentTeamId },
      }).then((response) => {
        expect(response.status).to.eq(401);
        expect(response.body).to.contain("Invalid room code");
      });
    });
  });

  it("keeps the mobile controller board compact, personalized, and sorted", () => {
    cy.viewport(390, 844);

    cy.request("/api/room").then(({ body: room }) => {
      pairPhone(room, "Ava", "Avocado Audibles").then((team) => {
        visitPairedPhone(room, team, "Ava");
      });
    });

    cy.contains("Phone Controller").should("be.visible");
    cy.get("#pick-players-grid").scrollIntoView();
    cy.get("#pick-players-grid").should("be.visible");
    cy.get("[data-player-card]").should("have.length.greaterThan", 2);
    cy.get("[data-player-card]").first().should("have.attr", "data-user-points");

    cy.get("[data-player-card]").then(($cards) => {
      const points = [...$cards].map((card) => Number(card.dataset.userPoints));
      const sorted = [...points].sort((left, right) => right - left);
      expect(points).to.deep.eq(sorted);
      expect(new Set(points).size).to.be.greaterThan(1);
    });

    cy.get("#pick-players-grid").then(($grid) => {
      const grid = $grid[0];
      const columns = getComputedStyle(grid).gridTemplateColumns.split(" ").filter(Boolean);
      expect(columns).to.have.length(1);

      const gridRect = grid.getBoundingClientRect();
      const visibleCards = [...grid.querySelectorAll("[data-player-card]")].filter((card) => {
        const rect = card.getBoundingClientRect();
        return rect.bottom > gridRect.top && rect.top < gridRect.bottom;
      });
      expect(visibleCards.length).to.be.within(1, 2);
    });
  });

  it("gives different paired phones different Jellycat point boards", () => {
    cy.viewport(390, 844);

    cy.request("/api/room").then(({ body: room }) => {
      pairPhone(room, "Ava", "Avocado Audibles").then((firstTeam) => {
        visitPairedPhone(room, firstTeam, "Ava");
        cy.get("[data-player-card]").first().should("have.attr", "data-user-points");
        cy.get("[data-player-card]").then(($firstCards) => {
          const firstBoard = [...$firstCards].slice(0, 6).map((card) => `${card.dataset.playerId}:${card.dataset.userPoints}`);

          pairPhone(room, "Bea", "Bunny Blitz").then((secondTeam) => {
            visitPairedPhone(room, secondTeam, "Bea");
            cy.get("[data-player-card]").first().should("have.attr", "data-user-points");
            cy.get("[data-player-card]").then(($secondCards) => {
              const secondBoard = [...$secondCards].slice(0, 6).map((card) => `${card.dataset.playerId}:${card.dataset.userPoints}`);
              expect(secondBoard).not.to.deep.eq(firstBoard);
            });
          });
        });
      });
    });
  });

  it("stores the personalized phone points on the drafted team roster", () => {
    cy.viewport(390, 844);

    cy.request("/api/room").then(({ body: room }) => {
      cy.request("/api/draft/state").then(({ body: state }) => {
        const team = state.teams.find((candidate) => candidate.id === state.currentTeamId);
        expect(team, "current team").to.exist;

        pairExistingPhone(room, "Casey", team).then((pairedTeam) => {
          visitPairedPhone(room, pairedTeam, "Casey");
        });

        cy.get("[data-player-card]").first().should("have.attr", "data-user-points");
        cy.get("[data-player-card]").first().then(($card) => {
          const playerId = $card[0].dataset.playerId;
          const personalizedPoints = Number($card[0].dataset.userPoints);

          cy.request("POST", "/api/draft/pick", {
            code: room.code,
            playerId,
            teamId: team.id,
          });

          cy.request("/api/draft/state").then(({ body: updatedState }) => {
            const updatedTeam = updatedState.teams.find((candidate) => candidate.id === team.id);
            const draftedPlayer = updatedTeam.players.find((player) => player.id === playerId);
            expect(draftedPlayer.points).to.eq(personalizedPoints);
          });
        });
      });
    });
  });

  it("shows anonymous Big Board with a scannable QR but no admin identity", () => {
    cy.visit("/draft");

    cy.contains("Room Display").should("be.visible");
    cy.contains("Controller Pairing").should("be.visible");
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