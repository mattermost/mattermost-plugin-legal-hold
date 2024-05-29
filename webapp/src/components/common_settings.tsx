import React, { useEffect, useMemo, useState } from 'react';

import { IntlProvider } from 'react-intl';

import SelectSetting, { OptionType } from './admin_console_settings/select_setting';

type CommonSettingsData = {
    TimeOfDay: string;
};

const useCommonSettingsForm = (initialValue: CommonSettingsData | undefined, onChange: (id: string, value: CommonSettingsData) => void) => {
    const [formState, setFormState] = useState<CommonSettingsData>({
        TimeOfDay: initialValue?.TimeOfDay ?? '',
    });

    return useMemo(() => ({
        formState,
        setFormValue: <T extends keyof CommonSettingsData>(key: T, value: CommonSettingsData[T]) => {
            const newState = {
                ...formState,
                [key]: value,
            };

            console.log(newState)

            setFormState(newState);
            onChange('PluginSettings.Plugins.com+mattermost+plugin-legal-hold.commonSettings', newState);
        },
    }), [formState, setFormState, onChange]);
};

type Props = {
    value: CommonSettingsData | undefined;
    onChange: (id: string, value: CommonSettingsData) => void;
};

const getJobTimeOptions = () => {
    const options: OptionType[] = [];
    return () => {
        if (options.length > 0) {
            return options;
        }
        const minuteIntervals = ['00', '15', '30', '45'];
        for (let h = 0; h < 24; h++) {
            let hourLabel = h;
            let hourValue = `${h}`;
            const timeOfDay = h >= 12 ? 'pm' : 'am';
            if (hourLabel < 10) {
                hourValue = `0${hourValue}`;
            }
            if (hourLabel > 12) {
                hourLabel -= 12;
            }
            if (hourLabel === 0) {
                hourLabel = 12;
            }
            for (let i = 0; i < minuteIntervals.length; i++) {
                options.push({
                    label: `${hourLabel}:${minuteIntervals[i]}${timeOfDay}`,
                    value: `${hourValue}:${minuteIntervals[i]}`
                });
            }
        }

        return options;
    };
};

const CommonSettings = (props: Props) => {
    const { formState, setFormValue } = useCommonSettingsForm(props.value, props.onChange);

    return (
        <IntlProvider locale='en-US'>
            <SelectSetting
                id='com.mattermost.plugin-legal-hold.TimeOfDay'
                name='Time of day'
                helpText='Time of day to run the Legal Hold task'
                value={formState.TimeOfDay}
                onChange={(value) => {
                    console.log("SelectSetting", value)
                    setFormValue('TimeOfDay', value)
                    console.log("config", formState)
                }}
                getOptions={getJobTimeOptions()}
            />
        </IntlProvider>
    );
};

export default CommonSettings;
