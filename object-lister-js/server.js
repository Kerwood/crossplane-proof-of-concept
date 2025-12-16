const express = require("express");
const session = require("express-session");
const { Storage } = require("@google-cloud/storage");
const { ConfidentialClientApplication } = require("@azure/msal-node");
const fs = require("fs");
const path = require("path");

const app = express();

//---[ Configuration ]-----------------
const BUCKET_NAME = process.env.BUCKET_NAME || "bucket-name-missing";
const REQUIRE_AUTH = process.env.REQUIRE_AUTH === "true";
const AZURE_CLIENT_ID = process.env.AZURE_CLIENT_ID;
const AZURE_TENANT_ID = process.env.AZURE_TENANT_ID;
const REDIRECT_URI = process.env.REDIRECT_URI ||
  "http://localhost:3000/auth/callback";
const SESSION_SECRET = process.env.SESSION_SECRET ||
  "your-secret-key-change-in-production";

if (REQUIRE_AUTH && (!AZURE_CLIENT_ID || !AZURE_TENANT_ID)) {
  throw new Error(
    "REQUIRE_AUTH is enabled but AZURE_CLIENT_ID or AZURE_TENANT_ID is not set.",
  );
}

//---[ Storage Client ]----------------
const storage = new Storage();

//---[ Session Setup ]-----------------
app.use(session({
  secret: SESSION_SECRET,
  resave: false,
  saveUninitialized: false,
  cookie: {
    secure: process.env.NODE_ENV === "production",
    httpOnly: true,
    maxAge: 24 * 60 * 60 * 1000, // 24 hours
  },
}));

//---[ MSAL Configuration ]------------
let msalClient = null;

if (REQUIRE_AUTH) {
  // Read the Kubernetes service account token
  const tokenPath = "/var/run/secrets/tokens/azure-token";
  let clientAssertion = "";

  try {
    clientAssertion = fs.readFileSync(tokenPath, "utf8").trim();
  } catch (err) {
    console.error(`Failed to read token from ${tokenPath}:`, err.message);
    throw new Error(
      "Workload identity token not found. Ensure the pod has the correct volume mount.",
    );
  }

  const msalConfig = {
    auth: {
      clientId: AZURE_CLIENT_ID,
      authority: `https://login.microsoftonline.com/${AZURE_TENANT_ID}`,
      clientAssertion: clientAssertion,
      clientAssertionType:
        "urn:ietf:params:oauth:client-assertion-type:jwt-bearer",
    },
    system: {
      loggerOptions: {
        loggerCallback: (level, message, containsPii) => {
          if (containsPii) return;
          console.log(message);
        },
        piiLoggingEnabled: false,
        logLevel: "Info",
      },
    },
  };

  msalClient = new ConfidentialClientApplication(msalConfig);
}

//---[ Authentication Middleware ]-----
const requireAuth = (req, res, next) => {
  if (!REQUIRE_AUTH) {
    return next();
  }

  if (req.session && req.session.isAuthenticated) {
    return next();
  }

  // Store the original URL to redirect after login
  req.session.returnTo = req.originalUrl;
  res.redirect("/auth/login");
};

//---[ Auth Routes ]-------------------
if (REQUIRE_AUTH) {
  // Login route - initiates the authorization code flow
  app.get("/auth/login", async (req, res) => {
    const authCodeUrlParameters = {
      scopes: ["user.read"],
      redirectUri: REDIRECT_URI,
    };

    try {
      const authCodeUrl = await msalClient.getAuthCodeUrl(
        authCodeUrlParameters,
      );
      res.redirect(authCodeUrl);
    } catch (error) {
      console.error("Error generating auth code URL:", error);
      res.status(500).send("Authentication error");
    }
  });

  // Callback route - handles the redirect from Azure AD
  app.get("/auth/callback", async (req, res) => {
    const tokenRequest = {
      code: req.query.code,
      scopes: ["user.read"],
      redirectUri: REDIRECT_URI,
    };

    try {
      // Re-read the token in case it has been refreshed by the kubelet
      const tokenPath = "/var/run/secrets/tokens/azure-token";
      const clientAssertion = fs.readFileSync(tokenPath, "utf8").trim();

      // Update client assertion for this request
      tokenRequest.clientAssertion = clientAssertion;
      tokenRequest.clientAssertionType =
        "urn:ietf:params:oauth:client-assertion-type:jwt-bearer";

      const response = await msalClient.acquireTokenByCode(tokenRequest);

      // Store user info in session
      req.session.isAuthenticated = true;
      req.session.account = response.account;

      // Redirect to original URL or home
      const returnTo = req.session.returnTo || "/";
      delete req.session.returnTo;
      res.redirect(returnTo);
    } catch (error) {
      console.error("Error acquiring token:", error);
      res.status(500).send("Authentication failed");
    }
  });

  // Logout route
  app.get("/auth/logout", (req, res) => {
    const account = req.session.account;
    req.session.destroy((err) => {
      if (err) {
        console.error("Error destroying session:", err);
      }

      // Redirect to Azure AD logout
      const logoutUri =
        `https://login.microsoftonline.com/${AZURE_TENANT_ID}/oauth2/v2.0/logout?post_logout_redirect_uri=${
          encodeURIComponent(REDIRECT_URI.replace("/auth/callback", "/"))
        }`;
      res.redirect(logoutUri);
    });
  });

  // User info endpoint (optional)
  app.get("/auth/user", requireAuth, (req, res) => {
    res.json({
      authenticated: true,
      user: req.session.account,
    });
  });
}

//---[ Protected Routes ]--------------
app.get("/", requireAuth, (_req, res) => {
  storage.bucket(BUCKET_NAME).getFiles()
    .then(([files]) => {
      const fileNames = files.map((file) => file.name);
      res.json({ bucket: BUCKET_NAME, files: fileNames });
    })
    .catch((err) => {
      console.error("Error listing objects:", err);
      res.status(500).json({ error: err.message });
    });
});

//---[ Health Check ]------------------
app.get("/health", (_req, res) => {
  res.json({ status: "ok", authRequired: REQUIRE_AUTH });
});

//---[ Start Server ]------------------
app.listen(3000, () => {
  console.log("Server is running on http://localhost:3000");
  console.log(`Authentication: ${REQUIRE_AUTH ? "ENABLED" : "DISABLED"}`);
});
