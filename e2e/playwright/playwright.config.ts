// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {PlaywrightTestConfig} from '@playwright/test';
import dotenv from 'dotenv';
import testConfig from '@e2e-test.playwright-config';
dotenv.config({path: `${__dirname}/.env`});

// Configuration override for plugin tests
testConfig.testDir = __dirname + '/tests';
testConfig.outputDir = __dirname + '/test-results';

const projects = testConfig.projects?.map((p) => ({...p, dependencies: ['setup']})) || [];
testConfig.projects = [{name: 'setup', testMatch: /test\.setup\.ts/} as PlaywrightTestConfig].concat(projects);
testConfig.use = {...testConfig.use, timezoneId: Intl.DateTimeFormat().resolvedOptions().timeZone};

export default testConfig;
