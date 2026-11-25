// Run the tests in a time zone behind UTC, where a date stored as midnight UTC falls on the previous local day.
/* eslint-disable no-process-env */
module.exports = async () => {
    process.env.TZ = 'America/Los_Angeles';
};
