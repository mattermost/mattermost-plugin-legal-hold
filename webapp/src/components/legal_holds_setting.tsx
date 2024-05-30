import React, {useEffect, useState} from 'react';

import {IntlProvider} from 'react-intl';

import Client from '@/client';
import {CreateLegalHold, LegalHold, UpdateLegalHold} from '@/types';

import CreateLegalHoldForm from '@/components/create_legal_hold_form';
import LegalHoldTable from '@/components/legal_hold_table';
import CreateLegalHoldButton from '@/components/create_legal_hold_button';

import LegalHoldIcon from '@/components/legal_hold_icon.svg';
import UpdateLegalHoldForm from '@/components/update_legal_hold_form';
import ConfirmRelease from '@/components/confirm_release';

const LegalHoldsSetting = () => {
    const [legalHoldsFetched, setLegalHoldsFetched] = useState(false);
    const [legalHoldsFetching, setLegalHoldsFetching] = useState(false);
    const [legalHolds, setLegalHolds] = useState(Array<LegalHold>());
    const [showCreateModal, setShowCreateModal] = useState(false);
    const [showUpdateModal, setShowUpdateModal] = useState(false);
    const [showReleaseModal, setShowReleaseModal] = useState(false);
    const [activeLegalHold, setActiveLegalHold] = useState<LegalHold|null>(null);

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
    });

    return (
        <IntlProvider locale='en-US'>
            <div
                style={{
                    padding: '28px 32px',
                    border: '1px solid rgba(var(--sys-center-channel-color-rgb), 0.08)',
                    background: 'var(--sys-center-channel-bg)',
                    borderRadius: '4px',
                    boxShadow: '0 2px 3px rgba(0, 0, 0, 0.08)',
                    marginBottom: '24px',
                }}
            >
                <div
                    style={{
                        display: 'flex',
                        alignItems: 'center',
                    }}
                >
                    <div
                        style={{
                            color: '#3f4350',
                            fontFamily: 'Metropolis',
                            fontSize: '18px',
                            fontWeight: '700',
                            lineHeight: '24px',
                            flexGrow: 1,
                        }}
                    >
                        {'Legal Holds'}
                    </div>
                    <CreateLegalHoldButton
                        onClick={() => setShowCreateModal(true)}
                    />
                </div>
                <hr/>

                {legalHolds.length === 0 && (
                    <div
                        style={{
                            display: 'flex',
                            flexDirection: 'column',
                            marginTop: '60px',
                            marginBottom: '60px',
                            justifyContent: 'center',
                            alignItems: 'center',
                        }}
                    >
                        <LegalHoldIcon/>
                        <p
                            style={{
                                fontSize: '20px',
                                fontWeight: 700,
                                marginBottom: 0,
                            }}
                        >
                            {'No legal holds'}
                        </p>
                        <p
                            style={{
                                paddingTop: '5px',
                                paddingBottom: '10px',
                            }}
                        >
                            {'You have no legal holds at the moment'}
                        </p>
                        <CreateLegalHoldButton
                            onClick={() => setShowCreateModal(true)}
                        />
                    </div>
                )}

                {legalHolds.length > 0 && (
                    <LegalHoldTable
                        legalHolds={legalHolds}
                        releaseLegalHold={doShowReleaseModal}
                        showUpdateModal={doShowUpdateModal}
                    />
                )}

                <CreateLegalHoldForm
                    createLegalHold={createLegalHold}
                    visible={showCreateModal}
                    onExited={() => setShowCreateModal(false)}
                />

                <UpdateLegalHoldForm
                    updateLegalHold={updateLegalHold}
                    visible={showUpdateModal}
                    onExited={() => setShowUpdateModal(false)}
                    legalHold={activeLegalHold}
                />

                <ConfirmRelease
                    legalHold={activeLegalHold}
                    releaseLegalHold={releaseLegalHold}
                    onExited={() => setShowReleaseModal(false)}
                    visible={showReleaseModal}
                />

            </div>
        </IntlProvider>
    );
};

export default LegalHoldsSetting;
