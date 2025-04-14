import { McpServer } from '@modelcontextprotocol/sdk/server/mcp.js';
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js';
import express from 'express';
import cors from 'cors';
import { z } from 'zod';
import { searchJobsHandler } from './handlers.js';
import logger from './logger.js';
import SseManager from './sseManager.js';


// Environment configuration
const PORT = process.env.JOBSPY_PORT || 9423;
const HOST = process.env.JOBSPY_HOST || '0.0.0.0';
const ENABLE_SSE = !!(process.env.ENABLE_SSE|0);

// Create the MCP server
const server = new McpServer({
  name: 'JobSpy MCP Server',
  version: '1.0.0',
  description:
    'A Model Context Protocol server that enables searching for jobs across various platforms',
});

const sseManager = new SseManager(server);

// Define tools for the MCP server
server.tool(
  'search_jobs',
  'Search for jobs across various job listing websites',
  {
    site_names: z
      .string()
      .describe(
        'Comma-separated list of job sites to search. Options: indeed,linkedin,zip_recruiter,glassdoor,google,bayt,naukri'
      )
      .default('indeed'),
    search_term: z
      .string()
      .describe('Search term for jobs')
      .default('software engineer'),
    location: z
      .string()
      .describe('Location for job search')
      .default('San Francisco, CA'),
    google_search_term: z
      .string()
      .nullable()
      .describe('Google specific search term')
      .default(null),
    results_wanted: z
      .number()
      .int()
      .describe('Number of results wanted')
      .default(20),
    hours_old: z
      .number()
      .int()
      .describe('How many hours old the jobs can be')
      .default(72),
    country_indeed: z
      .string()
      .describe('Country for Indeed search')
      .default('USA'),
    linkedin_fetch_description: z
      .boolean()
      .describe('Whether to fetch LinkedIn job descriptions (slower)')
      .default(false),
    proxies: z
      .string()
      .nullable()
      .describe('Comma-separated list of proxies')
      .default(null),
    format: z.enum(['json', 'csv']).describe('Output format').default('json'),
  },
  async (params, extra) => {
    let progressInterval;
    try {
      logger.info('Received search_jobs request', { params, extra });

      // Track progress for SSE clients
      if (sseManager.hasConnection(extra.sessionId)) {
        let progress = 0;
        progressInterval = setInterval(() => {
          progress += 5;
          if (progress > 90) {
            progress = 90; // Cap at 90% until complete
          }

          // Send progress to all connected clients
          sseManager.notificationProgress(
            {
              type: 'progress',
              tool: 'search_jobs',
              progress,
              message: `Searching for jobs (${progress}%)...`,
            },
            extra.sessionId
          );
        }, 2000);
      }

      // Execute job search
      const result = await searchJobsHandler(params);

      // Clean up progress interval
      if (progressInterval) {
        clearInterval(progressInterval);

        // Send 100% progress update to all connected clients
        if (sseManager.hasConnection(extra.sessionId)) {
          sseManager.notificationProgress(
            {
              type: 'progress',
              tool: 'search_jobs',
              progress: 100,
              message: 'Job search completed',
            },
            extra.sessionId
          );
        }
      }

      return result;
    } catch (error) {
      clearInterval(progressInterval);
      logger.error('Error in search_jobs handler', { error: error.message });
      return {
        isError: true,
        error: {
          message: error.message,
          code: 'INTERNAL_SERVER_ERROR',
        },
      };
    }
  }
);

// Initialize transports
let stdioTransport = null;
let httpServer = null;

// Start the server with configured transports
async function runServer() {
  logger.info('Starting JobSpy MCP server...');

  try {
    // Initialize and connect transports
    const connectedTransports = [];

    // Set up SSE transport if enabled
    if (ENABLE_SSE) {
      try {
        // Create Express app
        const app = express();

        // Configure CORS
        app.use(cors());

        // Configure Express middleware
        app.use(express.json());
        app.use(express.urlencoded({ extended: true }));

        // Health check endpoint
        app.get('/health', (req, res) => {
          res.status(200).json({ status: 'ok' });
        });

        // SSE endpoint for client connections
        app.get('/sse', async (req, res) => {
          const transport = sseManager.createTransport('/messages', res);

          res.on('close', () => {
            sseManager.removeTransport(transport.sessionId);
            logger.info(`Client disconnected: ${transport.sessionId}`);
          });

          await server.connect(transport);
          logger.info(`New SSE client connected: ${transport.sessionId}`);
        });

        // Message handling endpoint
        app.post('/messages', async (req, res) => {
          const transport = sseManager.getTransport(req);

          if (transport) {
            await transport.handlePostMessage(req, res, req.body);
          } else {
            res.status(400).send('No transport found for sessionId');
          }
        });

        app.post('/api', async (req, res) => {
            const data = searchJobsHandler(req.body);
            res.json(data).sendStatus(200);
        });

        // Start the Express server
        httpServer = app.listen(PORT, HOST, () => {
          logger.info(`SSE server listening at http://${HOST}:${PORT}`);
        });

        connectedTransports.push('SSE');

        logger.info(`SSE transport listening at http://${HOST}:${PORT}/sse`);
        logger.info(
          `Send endpoint available at http://${HOST}:${PORT}/messages`
        );
      } catch (error) {
        logger.error('Failed to connect SSE transport', {
          error: error.message,
          stack: error.stack,
        });
      }
    } else {
      // Set up stdio transport if no SSE
      try {
        stdioTransport = new StdioServerTransport();
        await server.connect(stdioTransport);
        connectedTransports.push('stdio');

        logger.info('Stdio transport connected');
      } catch (error) {
        logger.error('Failed to connect stdio transport', {
          error: error.message,
        });
      }
    }

    // Ensure at least one transport is connected
    if (connectedTransports.length === 0) {
      throw new Error('No transports connected. Check configuration.');
    }

    logger.info(
      `Server successfully connected with transports: ${connectedTransports.join(
        ', '
      )}`
    );
  } catch (error) {
    logger.error('Server connection error', {
      error: error.message,
      stack: error.stack,
    });
    process.exit(1);
  }
}

// Handle graceful shutdown
async function shutdown() {
  logger.info('Shutting down JobSpy MCP server...');

  try {
    // Disconnect all transports gracefully
    await server.disconnect();

    // Close HTTP server if it exists
    if (httpServer) {
      httpServer.close(() => {
        logger.info('HTTP server closed');
      });
    }

    logger.info('Server shutdown complete');
  } catch (error) {
    logger.error('Error during shutdown', { error: error.message });
  } finally {
    // Give logger time to flush
    setTimeout(() => process.exit(0), 100);
  }
}

process.on('SIGINT', shutdown);
process.on('SIGTERM', shutdown);

// Run the server
runServer().catch((error) => {
  logger.error('Unhandled error in server', { error: error.message });
  process.exit(1);
});
