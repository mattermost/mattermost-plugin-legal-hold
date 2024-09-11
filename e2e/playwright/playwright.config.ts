// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import dotenv from 'dotenv';
import testConfig from '@e2e-test.playwright-config';
dotenv.config({path: `${__dirname}/.env`});

// Configuration override for plugin tests
testConfig.testDir = __dirname + '/tests';
testConfig.outputDir = __dirname + '/test-results';

export default testConfig;
