import {Client4, ClientError} from '@mattermost/client';

import {CreateLegalHold, UpdateLegalHold} from '@/types';

import {manifest} from './manifest';

class APIClient {
    private readonly url = `/plugins/${manifest.id}/api/v1`;
    private readonly client4 = new Client4();

    downloadUrl = (id: string) => {
        return `${this.url}/legalhold/${id}/download`;
    };

    bundleUrl = (id: string) => {
        return `${this.url}/legalhold/${id}/bundle`;
    };

    getLegalHolds = () => {
        const url = `${this.url}/legalhold/list`;
        return this.doGet(url);
    };

    createLegalHold = (data: CreateLegalHold) => {
        const url = `${this.url}/legalhold/create`;
        return this.doPost(url, data);
    };

    releaseLegalHold = (id: string) => {
        const url = `${this.url}/legalhold/${id}/release`;
        return this.doPost(url, {});
    };

    updateLegalHold = (id: string, data: UpdateLegalHold) => {
        const url = `${this.url}/legalhold/${id}/update`;
        return this.doPost(url, data);
    };

    bundleLegalHold = (id: string) => {
        const url = this.bundleUrl(id);
        return this.doPost(url, {}) as Promise<{message: string}>;
    };

    testAmazonS3Connection = () => {
        const url = `${this.url}/test_amazon_s3_connection`;
        return this.doPost(url, {}) as Promise<{message: string}>;
    };

    private doGet = async (url: string, headers = {}) => {
        const options = {
            method: 'get',
            headers,
        };

        const response = await fetch(url, this.client4.getOptions(options));

        if (response.ok) {
            return response.json();
        }

        const text = await response.text();

        throw new ClientError(this.client4.url, {
            message: text || '',
            status_code: response.status,
            url,
        });
    };

    private doPost = async (url: string, body: any, headers = {}) => {
        const options = {
            method: 'post',
            body: JSON.stringify(body),
            headers,
        };

        const response = await fetch(url, this.client4.getOptions(options));

        if (response.ok) {
            return response.json();
        }

        const text = await response.text();

        throw new ClientError(this.client4.url, {
            message: text || '',
            status_code: response.status,
            url,
        });
    };
}

const Client = new APIClient();
export default Client;
