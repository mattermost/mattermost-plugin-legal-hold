import {useEffect, useState} from 'react';

import RefreshIcon from '@mattermost/compass-icons/components/refresh';
import {AdminPanel} from '@mattermost/compass-ui/components/admin-panel';
import {Button} from '@mattermost/compass-ui/components/button';
import {EmptyState} from '@mattermost/compass-ui/components/empty-state';
import {Icon} from '@mattermost/compass-ui/components/icon';
import {IconButton} from '@mattermost/compass-ui/components/icon-button';

import Client from '@/client';
import ConfirmRelease from '@/components/confirm_release';
import CreateLegalHoldForm from '@/components/create_legal_hold_form';
import LegalHoldTable from '@/components/legal_hold_table';
import ShowSecretModal from '@/components/show_secret_modal';
import UpdateLegalHoldForm from '@/components/update_legal_hold_form';
import type {CreateLegalHold, LegalHold, UpdateLegalHold} from '@/types';

import './legal_holds_setting.css';

const POLLING_INTERVAL = 30000; // 30 seconds in milliseconds

const LegalHoldsSetting = () => {
    const [legalHoldsFetched, setLegalHoldsFetched] = useState(false);
    const [legalHoldsFetching, setLegalHoldsFetching] = useState(false);
    const [legalHolds, setLegalHolds] = useState<LegalHold[]>([]);
    const [showCreateModal, setShowCreateModal] = useState(false);
    const [showUpdateModal, setShowUpdateModal] = useState(false);
    const [showReleaseModal, setShowReleaseModal] = useState(false);
    const [showSecretModal, setShowSecretModal] = useState(false);
    const [activeLegalHold, setActiveLegalHold] = useState<LegalHold|null>(null);

    const doRunLegalHold = async (id: string) => {
        try {
            const response = await Client.runLegalHold(id);
            return response;
        } catch (error) {
            console.log(error); //eslint-disable-line no-console
            throw error;
        }
    };

    const createLegalHold = async (data: CreateLegalHold) => {
        try {
            const response = await Client.createLegalHold(data);
            setLegalHoldsFetched(false);
            return response;
        } catch (error) {
            console.log(error); //eslint-disable-line no-console
            throw error;
        }
    };

    const releaseLegalHold = async (id: string) => {
        try {
            const response = await Client.releaseLegalHold(id);
            setLegalHoldsFetched(false);
            setActiveLegalHold(null);
            return response;
        } catch (error) {
            console.log(error); //eslint-disable-line no-console
            throw error;
        }
    };

    const updateLegalHold = async (data: UpdateLegalHold) => {
        try {
            const response = await Client.updateLegalHold(data.id, data);
            setLegalHoldsFetched(false);
            setActiveLegalHold(null);
            return response;
        } catch (error) {
            console.log(error); //eslint-disable-line no-console
            throw error;
        }
    };

    const doShowUpdateModal = (legalHold: LegalHold) => {
        setActiveLegalHold(legalHold);
        setShowUpdateModal(true);
    };

    const doShowReleaseModal = (legalHold: LegalHold) => {
        setActiveLegalHold(legalHold);
        setShowReleaseModal(true);
    };

    const doShowSecretModal = (legalHold: LegalHold) => {
        setActiveLegalHold(legalHold);
        setShowSecretModal(true);
    };

    useEffect(() => {
        const fetchLegalHolds = async () => {
            try {
                setLegalHoldsFetching(true);
                const data = await Client.getLegalHolds();
                setLegalHolds(data);
                setLegalHoldsFetching(false);
                setLegalHoldsFetched(true);
            } catch (error) {
                setLegalHoldsFetching(false);
                setLegalHoldsFetched(true);
                console.error(error); //eslint-disable-line no-console
            }
        };

        if (!legalHoldsFetched && !legalHoldsFetching) {
            fetchLegalHolds().catch(console.error); //eslint-disable-line no-console
        }
    }, [legalHoldsFetched, legalHoldsFetching]);

    // Set up polling
    useEffect(() => {
        const interval = setInterval(() => {
            setLegalHoldsFetched(false);
        }, POLLING_INTERVAL);

        return () => clearInterval(interval);
    }, []);

    // The header button is the primary action, so the one in the empty state is lower emphasis.
    const createButton = (testId: string, emphasis: 'primary' | 'secondary' = 'primary') => (
        <Button
            type='button'
            emphasis={emphasis}
            data-testid={testId}
            onClick={() => setShowCreateModal(true)}
        >
            {'Create new'}
        </Button>
    );

    return (
        <AdminPanel
            className='legal-hold-panel'
            title='Legal holds'
            headerActions={(
                <>
                    <IconButton
                        aria-label='Refresh legal holds'
                        title='Refresh legal holds'
                        icon={<Icon glyph={<RefreshIcon/>}/>}
                        onClick={() => setLegalHoldsFetched(false)}
                    />
                    {createButton('createNewLegalHoldOnTop')}
                </>
            )}
        >
            {legalHolds.length === 0 && (
                <EmptyState
                    title='No legal holds'
                    description='You have no legal holds at the moment'
                    actionSlot={createButton('createNewLegalHoldOnList', 'secondary')}
                />
            )}

            {legalHolds.length > 0 && (
                <LegalHoldTable
                    legalHolds={legalHolds}
                    releaseLegalHold={doShowReleaseModal}
                    showUpdateModal={doShowUpdateModal}
                    showSecretModal={doShowSecretModal}
                    runLegalHold={doRunLegalHold}
                    refresh={() => setLegalHoldsFetched(false)}
                />
            )}

            <CreateLegalHoldForm
                createLegalHold={createLegalHold}
                visible={showCreateModal}
                onClose={() => setShowCreateModal(false)}
            />

            <UpdateLegalHoldForm
                updateLegalHold={updateLegalHold}
                visible={showUpdateModal}
                onClose={() => setShowUpdateModal(false)}
                legalHold={activeLegalHold}
            />

            <ShowSecretModal
                legalHold={activeLegalHold}
                visible={showSecretModal}
                onClose={() => setShowSecretModal(false)}
            />

            <ConfirmRelease
                legalHold={activeLegalHold}
                releaseLegalHold={releaseLegalHold}
                onClose={() => setShowReleaseModal(false)}
                visible={showReleaseModal}
            />
        </AdminPanel>
    );
};

export default LegalHoldsSetting;
