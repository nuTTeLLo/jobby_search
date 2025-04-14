require('dotenv').config();
const express = require('express');
const morgan = require('morgan');
const { searchJobsHandler } = require('./handlers');
const logger = require('./logger');

const app = express();
const PORT = process.env.PORT || 9423;

// Log configuration on startup
logger.info('Server starting with configuration', {
  port: PORT,
  dockerImage: process.env.JOBSPY_DOCKER_IMAGE || 'jobspy',
  accessTokenConfigured: !!process.env.JOBSPY_ACCESS_TOKEN
});

app.use(morgan('combined'));
app.use(express.json());

app.get('/search', searchJobsHandler);

app.listen(PORT, () => {
  logger.info(`Server is running on port ${PORT}`);
});
